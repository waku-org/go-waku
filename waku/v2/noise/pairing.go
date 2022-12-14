package noise

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"sync"
	"time"

	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	n "github.com/waku-org/noise"
	"go.uber.org/zap"
)

var ErrPairingTimeout = errors.New("pairing has timed out")


type Sender interface {
	Publish(msg *pb.WakuMessage) <-chan struct{}
}

type Receiver {
	subscribe(decoder: Decoder<NoiseHandshakeMessage>): Promise<void>;

	// next message should return messages received in a content topic
	// messages should be kept in a queue, meaning that nextMessage
	// will call pop in the queue to remove the oldest message received
	// (it's important to maintain order of received messages)
	nextMessage(contentTopic: string): Promise<NoiseHandshakeMessage>;
}

type Pairing struct {
	sync.RWMutex

	ContentTopic string

	randomFixLenVal      []byte
	myCommittedStaticKey []byte

	params    PairingParameters
	handshake *Handshake

	authCode          string
	authCodeEmitted   chan string
	authCodeConfirmed chan bool

	timeoutCh chan struct{}

	logger *zap.Logger

	started bool
}

type PairingParameterOption func(*PairingParameters) error

func WithInitiatorParameters(qrString string, qrMessageNametag MessageNametag) PairingParameterOption {
	return func(params *PairingParameters) error {
		params.initiator = true
		qr, err := StringToQR(qrString)
		if err != nil {
			return err
		}
		params.qr = qr
		params.qrMessageNametag = qrMessageNametag
		return nil
	}
}
func WithReceiverParameters(applicationName, applicationVersion, shardId string, myEphemeralPublicKey ed25519.PublicKey, qrMessageNameTag *MessageNametag) PairingParameterOption {
	return func(params *PairingParameters) error {
		params.initiator = false
		if qrMessageNameTag == nil {
			b := make([]byte, MessageNametagLength)
			_, err := rand.Read(b)
			if err != nil {
				return err
			}
			params.qrMessageNametag = BytesToMessageNametag(b)
		} else {
			params.qrMessageNametag = *qrMessageNameTag
			params.qr = NewQR(applicationName, applicationVersion, shardId, myEphemeralPublicKey, params.myCommitedStaticKey)
		}
		return nil
	}
}

type PairingParameters struct {
	myCommitedStaticKey []byte
	initiator           bool
	qr                  QR
	qrMessageNametag    MessageNametag
}

func NewPairing(myStaticKey n.DHKey, myEphemeralKey n.DHKey, opts PairingParameterOption, logger *zap.Logger) (*Pairing, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	myCommitedStaticKey := CommitPublicKey(myStaticKey.Public, b)

	var params PairingParameters
	params.myCommitedStaticKey = myCommitedStaticKey
	opts(&params)

	preMessagePKs := params.qr.ephemeralPublicKey
	hs, err := NewHandshake_WakuPairing_25519_ChaChaPoly_SHA256(myStaticKey, myEphemeralKey, params.initiator, params.qr.Bytes(), preMessagePKs)
	if err != nil {
		return nil, err
	}

	return &Pairing{
		ContentTopic:         "/" + params.qr.applicationName + "/" + params.qr.applicationVersion + "/wakunoise/1/sessions_shard-" + params.qr.shardId + "/proto",
		randomFixLenVal:      b, // r or s depending if you're responder or initiator
		myCommittedStaticKey: myCommitedStaticKey,
		authCodeEmitted:      make(chan string, 1),
		authCodeConfirmed:    make(chan bool, 1),
		params:               params,
		handshake:            hs,
		logger:               logger.Named("waku-pairing-1"),
	}, nil
}

func (p *Pairing) PairingInfo() (string, MessageNametag) {
	p.RLock()
	defer p.RUnlock()
	return p.params.qr.String(), p.params.qrMessageNametag
}

func (p *Pairing) Execute(timeout time.Duration) error {
	p.RLock()
	if p.started {
		p.RUnlock()
		return errors.New("pairing already executed. Create new pairing object")
	}

	p.RUnlock()
	p.Lock()
	p.started = true
	p.timeoutCh = make(chan struct{}, 1)
	p.Unlock()

	t := time.NewTimer(timeout)
	defer t.Stop()

	var doneCh <-chan error
	if p.params.initiator {
		doneCh = p.initiatorHandshake()
	} else {
		doneCh = p.responderHandshake()
	}

	select {
	case <-t.C:
		p.Lock()
		defer p.Unlock()
		close(p.timeoutCh)
		return ErrPairingTimeout
	case err := <-doneCh:
		return err
	}
}

func (p *Pairing) isAuthCodeConfirmed() (bool, error) {
	// wait for user to confirm or not, or for the whole pairing process to time out
	select {
	case <-p.timeoutCh:
		return false, ErrPairingTimeout
	case confirmed := <-p.authCodeConfirmed:
		return confirmed, nil
	}
}

/*
func executeReadStepWithNextMessage(contentTopic string, messageNametag MessageNameTag) error {
    // TODO: create test unit for this function
    let stopLoop = false;

    this.eventEmitter.once("pairingTimeout", () => {
      stopLoop = true;
    });

    this.eventEmitter.once("pairingComplete", () => {
      stopLoop = true;
    });

    while (!stopLoop) {
      try {
        const hsMessage = await this.receiver.nextMessage(contentTopic);
        const step = this.handshake.stepHandshake({
          readPayloadV2: hsMessage.payloadV2,
          messageNametag,
        });
        return step;
      } catch (err) {
        if (err instanceof MessageNametagError) {
          console.debug("Unexpected message nametag", err.expectedNametag, err.actualNametag);
        } else {
          throw err;
        }
      }
    }

    throw new Error("could not obtain next message");
  }
*/

func (p *Pairing) initiatorHandshake() (doneCh chan error) {
	doneCh = make(chan error, 1)

	go func() {
		<-p.receiver.subscribe(p.ContentTopic)

		// The handshake initiator writes a Waku2 payload v2 containing the handshake message
		// and the (encrypted) transport message
		// The message is sent with a messageNametag equal to the one received through the QR code
		hsStep, err := p.handshake.Step(nil, p.myCommittedStaticKey, &p.params.qrMessageNametag)
		if err != nil {
			doneCh <- err
			return
		}
		// We prepare a message from initiator's payload2
		// At this point wakuMsg is sent over the Waku network to receiver content topic
		<-p.sender.publish(p.ContentTopic, hsStep)

		// We generate an authorization code using the handshake state
		// this check has to be confirmed with a user interaction, comparing auth codes in both ends

		authCode, err := p.handshake.Authcode()
		if err != nil {
			doneCh <- err
			return
		}

		p.authCodeEmitted <- authCode

		p.logger.Info("waiting for authcode confirmation....")

		confirmed, err := p.isAuthCodeConfirmed()
		if err != nil {
			doneCh <- err
			return
		}
		if !confirmed {
			p.logger.Info("authcode not confirmed")
		}

		close(doneCh)
		/*
		   // 2nd step
		   // <- sB, eAsB    {r}
		   hsStep = await this.executeReadStepWithNextMessage(this.contentTopic, this.handshake.hs.toMessageNametag());

		   if (!this.handshake.hs.rs) throw new Error("invalid handshake state");

		   // Initiator further checks if receiver's commitment opens to receiver's static key received
		   const expectedReceiverCommittedStaticKey = commitPublicKey(this.handshake.hs.rs, hsStep.transportMessage);
		   if (!uint8ArrayEquals(expectedReceiverCommittedStaticKey, this.qr.committedStaticKey)) {
		     throw new Error("expected committed static key does not match the receiver actual committed static key");
		   }

		   // 3rd step
		   // -> sA, sAeB, sAsB  {s}
		   // Similarly as in first step, the initiator writes a Waku2 payload containing the handshake message and the (encrypted) transport message
		   hsStep = this.handshake.stepHandshake({
		     transportMessage: this.randomFixLenVal,
		     messageNametag: this.handshake.hs.toMessageNametag(),
		   });

		   encoder = new NoiseHandshakeEncoder(this.contentTopic, hsStep);
		   await this.sender.publish(encoder, {});

		   // Secure Transfer Phase
		   this.handshakeResult = this.handshake.finalizeHandshake();

		   this.eventEmitter.emit("pairingComplete");

		   return WakuPairing.getSecureCodec(this.contentTopic, this.handshakeResult);*/
	}()

	return doneCh
}

func (p *Pairing) responderHandshake() (doneCh chan error) {
	doneCh = make(chan error, 1)

	func() {
		<-p.receiver.subscribe(p.ContentTopic)

		close(doneCh)

		/*
			    // the received reads the initiator's payloads, and returns the (decrypted) transport message the initiator sent
			    // Note that the received verifies if the received payloadV2 has the expected messageNametag set
			    let hsStep = await this.executeReadStepWithNextMessage(this.contentTopic, this.qrMessageNameTag);

			    const initiatorCommittedStaticKey = new Uint8Array(hsStep.transportMessage);

			    const confirmationPromise = this.isAuthCodeConfirmed();
			    await delay(100);
			    this.eventEmitter.emit("authCodeGenerated", this.handshake.genAuthcode());
			    console.log("Waiting for authcode confirmation...");
			    const confirmed = await confirmationPromise;
			    if (!confirmed) {
			      throw new Error("authcode is not confirmed");
			    }


				/*
			    // 2nd step
			    // <- sB, eAsB    {r}
			    // Receiver writes and returns a payload
			    hsStep = this.handshake.stepHandshake({
			      transportMessage: this.randomFixLenVal,
			      messageNametag: this.handshake.hs.toMessageNametag(),
			    });

			    // We prepare a Waku message from receiver's payload2
			    const encoder = new NoiseHandshakeEncoder(this.contentTopic, hsStep);
			    await this.sender.publish(encoder, {});

			    // 3rd step
			    // -> sA, sAeB, sAsB  {s}

			    // The receiver reads the initiator's payload sent by the initiator
			    hsStep = await this.executeReadStepWithNextMessage(this.contentTopic, this.handshake.hs.toMessageNametag());

			    if (!this.handshake.hs.rs) throw new Error("invalid handshake state");

			    // The receiver further checks if the initiator's commitment opens to the initiator's static key received
			    const expectedInitiatorCommittedStaticKey = commitPublicKey(this.handshake.hs.rs, hsStep.transportMessage);
			    if (!uint8ArrayEquals(expectedInitiatorCommittedStaticKey, initiatorCommittedStaticKey)) {
			      throw new Error("expected committed static key does not match the initiator actual committed static key");
			    }

			    // Secure Transfer Phase
			    this.handshakeResult = this.handshake.finalizeHandshake();

			    this.eventEmitter.emit("pairingComplete");

			    return WakuPairing.getSecureCodec(this.contentTopic, this.handshakeResult);*/
	}()
	return doneCh
}

func (p *Pairing) ConfirmAuthCode(confirmed bool) error {
	p.Lock()
	authcode := p.authCode
	p.Unlock()

	if authcode != "" {
		p.authCodeConfirmed <- confirmed

		return nil
	}

	return errors.New("authcode has not been generated yet")
}

func (p *Pairing) AuthCode() chan<- string {
	ch := make(chan string, 1)

	p.Lock()
	authcode := p.authCode
	p.Unlock()

	if authcode != "" {
		ch <- authcode
	} else {
		ch <- <-p.authCodeEmitted
	}
	close(ch)
	return ch
}
