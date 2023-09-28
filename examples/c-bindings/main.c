#include <stdbool.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <time.h>
#include <stdint.h>
#include <inttypes.h>

#include "libgowaku.h"
#include "nxjson.c"
#include "base64.h"
#include "main.h"

char *alicePrivKey = "0x4f012057e1a1458ce34189cb27daedbbe434f3df0825c1949475dec786e2c64e";
char *alicePubKey = "0x0440f05847c4c7166f57ae8ecaaf72d31bddcbca345e26713ca9e26c93fb8362ddcd5ae7f4533ee956428ad08a89cd18b234c2911a3b1c7fbd1c0047610d987302";

char *bobPrivKey = "0xb91d6b2df8fb6ef8b53b51b2b30a408c49d5e2b530502d58ac8f94e5c5de1453";
char *bobPubKey = "0x045eef61a98ba1cf44a2736fac91183ea2bd86e67de20fe4bff467a71249a8a0c05f795dd7f28ced7c15eaa69c89d4212cc4f526ca5e9a62e88008f506d850cccd";


char* result = NULL;
char* contentTopic;

void handle_ok(const char* msg, size_t len) {
    if (result != NULL) {
        free(result);
    }
    result = malloc(len * sizeof(char) + 1);
    strcpy(result, msg);
}

void handle_error(const char* msg, size_t len) {
    printf("Error: %s\n", msg);
    exit(1);
}

void callBack(const char* signal, size_t len_0)
{
  // This callback will be executed each time a new message is received

  // Example signal:
  /*{
      "nodeId":1,
      "type":"message",
      "event":{
        "pubsubTopic":"/waku/2/default-waku/proto",
        "messageID":"0x6496491e40dbe0b6c3a2198c2426b16301688a2daebc4f57ad7706115eac3ad1",
        "wakuMessage":{
          "payload":"BPABASUqWgRkgp73aW/FHIyGtJDYnStvaQvCoX9MdaNsOH39Vet0em6ipZc3lZ7kK9uFFtbJgIWfRaqTxSRjiFOPx88gXt1JeSm2SUwGSz+1gh2xTy0am8tXkc8OWSSjamdkEbXuVgAueLxHOnV3xlGwYt7nx2G5DWYqUu1BXv4yWHPOoiH2yx3fxX0OajgKGBwiMbadRNUuAUFPRM90f+bzG2y22ssHctDV/U6sXOa9ljNgpAx703Q3WIFleSRozto7ByNAdRFwWR0RGGV4l0btJXM7JpnrYcVC24dB0tJ3HVWuD0ZcwOM1zTL0wwc0hTezLHvI+f6bHSzsFGcCWIlc03KSoMjK1XENNL4dtDmSFI1DQCGgq09c2Bc3Je3Ci6XJHu+FP1F1pTnRzevv2WP8FSBJiTXpmJXdm6evB7V1Xxj4QlzQDvmHLRpBOL6PSttxf1Dc0IwC6BfZRN5g0dNmItNlS2pcY1MtZLxD5zpj",
          "contentTopic":"ABC",
          "version":1,
          "timestamp":1647826358000000000
        }
      }
    }*/

  const nx_json *json = nx_json_parse((char*) signal, 0);
  const char *type = nx_json_get(json, "type")->text_value;

  if (strcmp(type, "message") == 0)
  {
    const nx_json *wakuMsgJson = nx_json_get(nx_json_get(json, "event"), "wakuMessage");
    const char *contentTopic = nx_json_get(wakuMsgJson, "contentTopic")->text_value;

    if (strcmp(contentTopic, contentTopic) == 0)
    {
      char* msg = utils_extract_wakumessage_from_signal(wakuMsgJson);
      WAKU_CALL(waku_decode_asymmetric(msg, bobPrivKey, handle_ok, handle_error));

      char *decodedMsg = strdup(result);

      const nx_json *dataJson = nx_json_parse(decodedMsg, 0);

      const char *pubkey = nx_json_get(dataJson, "pubkey")->text_value;
      const char *base64data = nx_json_get(dataJson, "data")->text_value;
          
      size_t data_len = b64_decoded_size(base64data);
      char *data = malloc(data_len);
    
      b64_decode(base64data,  (unsigned char *)data, data_len) ;

      printf(">>> Received \"%s\" from %s\n", data, pubkey);
      fflush(stdout);
    }
  }

  nx_json_free(json);
}

int main(int argc, char *argv[])
{
  waku_set_event_callback(callBack);

  char *configJSON = "{\"host\": \"0.0.0.0\", \"port\": 60000, \"logLevel\":\"error\", \"store\":true}";
  WAKU_CALL( waku_new(configJSON, handle_error) );  // configJSON can be NULL too to use defaults

  WAKU_CALL(waku_start(handle_error) ); // Start the node, enabling the waku protocols

  WAKU_CALL(waku_peerid(handle_ok, handle_error)); // Obtain the node peerID
  char *peerID = strdup(result);
  printf("PeerID: %s\n", result);

  WAKU_CALL(waku_content_topic("example", 1, "default", "rfc26", handle_ok));
  contentTopic = strdup(result);
  
  WAKU_CALL(waku_connect("/dns4/node-01.gc-us-central1-a.wakuv2.test.statusim.net/tcp/30303/p2p/16Uiu2HAmJb2e28qLXxT5kZxVUUoJt72EMzNGXB47Rxx5hw3q4YjS", 0, handle_error)); // Connect to a node

  // To use dns discovery, and retrieve nodes from a enrtree url
 WAKU_CALL( waku_dns_discovery("enrtree://AO47IDOLBKH72HIZZOXQP6NMRESAN7CHYWIBNXDXWRJRZWLODKII6@test.wakuv2.nodes.status.im", "", 0, handle_ok, handle_error)); // Discover Nodes
  printf("Discovered nodes: %s\n", result);
  
  WAKU_CALL(waku_default_pubsub_topic(handle_ok));
  char *pubsubTopic = strdup(result);
  printf("Default pubsub topic: %s\n", pubsubTopic);

  // To see a store query in action:
  /*
  char query[1000];
  sprintf(query, "{\"pubsubTopic\":\"%s\", \"pagingOptions\":{\"pageSize\": 40, \"forward\":false}}", pubsubTopic);
  WAKU_CALL(waku_store_query(query, NULL, 0, handle_ok, handle_error));
  printf("%s\n",result);
  */

  WAKU_CALL( waku_relay_subscribe(NULL, handle_error));

  int i = 0;
  int version = 1;
  while (i < 5)
  {
    i++;

    char wakuMsg[1000];
    char *msgPayload = b64_encode("Hello World!", 12);

    sprintf(wakuMsg, "{\"payload\":\"%s\",\"contentTopic\":\"%s\",\"timestamp\":%"PRIu64"}", msgPayload, contentTopic, nowInNanosecs());
    free(msgPayload);


    WAKU_CALL(waku_encode_asymmetric(wakuMsg, bobPubKey, alicePrivKey, handle_ok, handle_error));
    char *encodedMessage = strdup(result);

    WAKU_CALL(waku_relay_publish(encodedMessage, NULL, 0, handle_ok, handle_error)); // Broadcast via waku relay
    printf("C\n");

    char *messageID = strdup(result);
    printf("MessageID: %s\n",messageID);

    free(messageID);

    sleep(1);
  }

    
  // To retrieve messages from local store, set store:true in the node config, and use waku_store_local_query
  /*
  char query[1000];
  sprintf(query, "{\"pubsubTopic\":\"%s\", \"pagingOptions\":{\"pageSize\": 40, \"forward\":false}}", pubsubTopic);
  WAKU_CALL(waku_store_local_query(query, handle_ok, handle_error));
  printf("%s\n",result);
  */

  WAKU_CALL(waku_stop(handle_error));
  
  return 0;
}
