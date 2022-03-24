using System.Runtime.InteropServices;
using System.Text.Json;

namespace Waku
{
    internal static class Constants
    {
        public const string dllName = "libs/libgowaku.dll";
    }
    internal static class Response
    {
        [DllImport(Constants.dllName)]
        internal static extern IntPtr gowaku_utils_free(IntPtr ptr);

        internal static string PtrToStringUtf8(IntPtr ptr) // aPtr is nul-terminated
        {
            if (ptr == IntPtr.Zero)
            {
                gowaku_utils_free(ptr);
                return "";
            }

            int len = 0;
            while (System.Runtime.InteropServices.Marshal.ReadByte(ptr, len) != 0)
                len++;

            if (len == 0)
            {
                gowaku_utils_free(ptr);
                return "";
            }

            byte[] array = new byte[len];
            System.Runtime.InteropServices.Marshal.Copy(ptr, array, 0, len);
            string result = System.Text.Encoding.UTF8.GetString(array);
            gowaku_utils_free(ptr);
            return result;
        }
        internal static T HandleResponse<T>(IntPtr ptr, string errNoValue) where T : struct, IComparable
        {
            string strResponse = PtrToStringUtf8(ptr);

            JsonResponse<T?>? response = JsonSerializer.Deserialize<JsonResponse<T?>>(strResponse);

            if (response == null) throw new Exception("unknown waku error");

            if (response.error != null) throw new Exception(response.error);

            if (!response.result.HasValue) throw new Exception(errNoValue);

            return response.result.Value;
        }

        internal static void HandleResponse(IntPtr ptr)
        {
            string strResponse = PtrToStringUtf8(ptr);

            JsonResponse<string>? response = JsonSerializer.Deserialize<JsonResponse<string>>(strResponse);

            if (response == null) throw new Exception("unknown waku error");

            if (response.error != null) throw new Exception(response.error);
        }

        internal static string HandleResponse(IntPtr ptr, string errNoValue)
        {
            string strResponse = PtrToStringUtf8(ptr);

            JsonResponse<string>? response = JsonSerializer.Deserialize<JsonResponse<string>>(strResponse);

            if (response == null) throw new Exception("unknown waku error");

            if (response.error != null) throw new Exception(response.error);

            if (String.IsNullOrEmpty(response.result)) throw new Exception(errNoValue);

            return response.result;
        }

        internal static IList<T> HandleListResponse<T>(IntPtr ptr, string errNoValue)
        {
            string strResponse = PtrToStringUtf8(ptr);
            Console.WriteLine(strResponse);
            JsonResponse<IList<T>>? response = JsonSerializer.Deserialize<JsonResponse<IList<T>>>(strResponse);

            if (response == null) throw new Exception("unknown waku error");

            if (response.error != null) throw new Exception(response.error);

            if (response.result == null) throw new Exception(errNoValue);

            return response.result;
        }
    }

    public class Config
    {
        public string? host { get; set; }
        public int? port { get; set; }
        public string? advertiseAddr { get; set; }
        public string? nodeKey { get; set; }
        public int? keepAliveInterval { get; set; }
        public bool? relay { get; set; }
    }

    public static class Utils
    {
        [DllImport(Constants.dllName)]
        internal static extern IntPtr gowaku_default_pubsub_topic();

        /// <summary>
        /// Get default pubsub topic
        /// </summary>
        /// <returns>Default pubsub topic used for exchanging waku messages defined in RFC 10</returns>
        public static string DefaultPubsubTopic()
        {
            IntPtr ptr = gowaku_default_pubsub_topic();
            return Response.PtrToStringUtf8(ptr);
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr gowaku_content_topic(string applicationName, uint applicationVersion, string contentTopicName, string encoding);

        /// <summary>
        /// Create a content topic string
        /// </summary>
        /// <param name="applicationName"></param>
        /// <param name="applicationVersion"></param>
        /// <param name="contentTopicName"></param>
        /// <param name="encoding"></param>
        /// <returns>Content topic string according to RFC 23</returns>
        public static string ContentTopic(string applicationName, uint applicationVersion, string contentTopicName, string encoding)
        {
            IntPtr ptr = gowaku_content_topic(applicationName, applicationVersion, contentTopicName, encoding);
            return Response.PtrToStringUtf8(ptr);
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr gowaku_pubsub_topic(string name, string encoding);

        /// <summary>
        /// Create a pubsub topic string
        /// </summary>
        /// <param name="name"></param>
        /// <param name="encoding"></param>
        /// <returns>Pubsub topic string according to RFC 23</returns>
        public static string PubsubTopic(string name, string encoding)
        {
            IntPtr ptr = gowaku_pubsub_topic(name, encoding);
            return Response.PtrToStringUtf8(ptr);
        }
    }

    internal class JsonResponse<T>
    {
        public string? error { get; set; }
        public T? result { get; set; }
    }

    public class Node
    {
        private int _nodeId;
        private bool _running;

        [DllImport(Constants.dllName)]
        internal static extern IntPtr gowaku_new(string configJSON);

        /// <summary>
        /// Initialize a waku node.
        /// </summary>
        /// <param name="c">Waku.Config containing the options used to initialize a go-waku node. It can be NULL to use defaults. All the keys from the configuration are optional </param>
        /// <returns>The node id</returns>
        public Node(Config? c = null)
        {
            string jsonString = "{}";
            if (c != null)
            {
                jsonString = JsonSerializer.Serialize(c);
            }

            IntPtr ptr = gowaku_new(jsonString);
            _nodeId = Response.HandleResponse<int>(ptr, "could not obtain the node id");
        }

        ~Node()
        {
            if (_running) {
                Stop();
            }
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr gowaku_start(int nodeId);

        /// <summary>
        /// Initialize a go-waku node mounting all the protocols that were enabled during the waku node initialization.
        /// </summary>
        public void Start()
        {
            IntPtr ptr = gowaku_start(_nodeId);
            Response.HandleResponse(ptr);

            _running = true;
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr gowaku_stop(int nodeId);

        /// <summary>
        /// Stops a go-waku node
        /// </summary>
        public void Stop()
        {
            IntPtr ptr = gowaku_stop(_nodeId);
            Response.HandleResponse(ptr);

            _running = false;
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr gowaku_peerid(int nodeId);

        /// <summary>
        /// Obtain the peer ID of the go-waku node.
        /// </summary>
        /// <returns>The base58 encoded peer Id</returns>
        public string PeerId()
        {
            IntPtr ptr = gowaku_peerid(_nodeId);
            return Response.HandleResponse(ptr, "could not obtain the peerId");
        }

        [DllImport(Constants.dllName)]
        internal static extern IntPtr gowaku_peer_cnt(int nodeId);

        /// <summary>
        /// Obtain number of connected peers
        /// </summary>
        /// <returns>The number of peers connected to this node</returns>
        public int PeerCnt()
        {
            IntPtr ptr = gowaku_start(_nodeId);
            return Response.HandleResponse<int>(ptr, "could not obtain the peer count");
        }


        [DllImport(Constants.dllName)]
        internal static extern IntPtr gowaku_listen_addresses(int nodeId);

        /// <summary>
        /// Obtain the multiaddresses the wakunode is listening to
        /// </summary>
        /// <returns>List of multiaddresses</returns>
        public IList<string> ListenAddresses()
        {
            IntPtr ptr = gowaku_listen_addresses(_nodeId);
            return Response.HandleListResponse<string>(ptr, "could not obtain the peer count");
        }


        /*
        // Add node multiaddress and protocol to the wakunode peerstore
        extern __declspec(dllexport) char* gowaku_add_peer(int nodeID, char* address, char* protocolID);

        // Dial peer at multiaddress. if ms > 0, cancel the function execution if it takes longer than N milliseconds
        extern __declspec(dllexport) char* gowaku_dial_peer(int nodeID, char* address, int ms);

        // Dial known peer by peerID. if ms > 0, cancel the function execution if it takes longer than N milliseconds
        extern __declspec(dllexport) char* gowaku_dial_peerid(int nodeID, char* peerID, int ms);

*/

        [DllImport(Constants.dllName)]
        internal static extern IntPtr gowaku_close_peerid(int nodeId, string peerID);

        // Close connection to a known peer by peerID
        public void ClosePeer(string peerId)
        {
            IntPtr ptr = gowaku_close_peerid(peerID);
            Response.HandleResponse(ptr);
        }


        // Publish a message using waku relay. Use NULL for topic to use the default pubsub topic
        // If ms is greater than 0, the broadcast of the message must happen before the timeout
        // (in milliseconds) is reached, or an error will be returned
        [DllImport(Constants.dllName)]
        internal static extern IntPtr gowaku_relay_publish(int nodeId, string messageJSON, string topic, int ms);

        
        



// Determine if there are enough peers to publish a message on a topic. Use NULL
        // to verify the number of peers in the default pubsub topic
        [DllImport(Constants.dllName)]
        internal static extern bool gowaku_enough_peers(int nodeId, string topic);

        




    void MyEventHandler(IntPtr signalJSON)
    {
        // TODO: call another callback that receives a string if it's not null
        Console.WriteLine(PtrToStringUtf8(signalJSON));
    }


    delegate void EventHandler(IntPtr signalJSON);

    // Register callback to act as signal handler and receive application signal
    // (in JSON) which are used o react to asyncronous events in waku. The function
    // signature for the callback should be `void myCallback(char* signalJSON)`
    [DllImport(Constants.dllName, CallingConvention = CallingConvention.Cdecl)]
    internal static extern void gowaku_set_event_callback(EventHandler cb);


    public SetEventCallback() // Call this on constructor
    {
        gowaku_set_event_callback(MyEventHandler);
    }  




        // Subscribe to a WakuRelay topic. Set the topic to NULL to subscribe
        // to the default topic. Returns a json response containing the subscription ID
        // or an error message. When a message is received, a "message" is emitted containing
        // the message, pubsub topic, and nodeID in which the message was received
        [DllImport(Constants.dllName)]
        internal static extern string gowaku_relay_subscribe(int nodeId, string topic);

        // Closes the pubsub subscription to a pubsub topic. Existing subscriptions
        // will not be closed, but they will stop receiving messages
        [DllImport(Constants.dllName)]
        internal static extern string gowaku_relay_unsubscribe_from_topic(int nodeId, string topic);

        // Closes a waku relay subscription
        [DllImport(Constants.dllName)]
        internal static extern string gowaku_relay_close_subscription(int nodeId, string subsID);


/*
        // Retrieve the list of peers known by the waku node
        extern __declspec(dllexport) char* gowaku_peers(int nodeID);

        // Encode a byte array. `keyType` defines the type of key to use: `NONE`,
        // `ASYMMETRIC` and `SYMMETRIC`. `version` is used to define the type of
        // payload encryption:
        // When `version` is 0
        // - No encryption is used
        // When `version` is 1
        // - If using `ASYMMETRIC` encoding, `key` must contain a secp256k1 public key
        //   to encrypt the data with,
        // - If using `SYMMETRIC` encoding, `key` must contain a 32 bytes symmetric key.
        // The `signingKey` can contain an optional secp256k1 private key to sign the
        // encoded message, otherwise NULL can be used.
        extern __declspec(dllexport) char* gowaku_encode_data(char* data, char* keyType, char* key, char* signingKey, int version);

        // Decode a byte array. `keyType` defines the type of key used: `NONE`,
        // `ASYMMETRIC` and `SYMMETRIC`. `version` is used to define the type of
        // encryption that was used in the payload:
        // When `version` is 0
        // - No encryption was used. It will return the original message payload
        // When `version` is 1
        // - If using `ASYMMETRIC` encoding, `key` must contain a secp256k1 public key
        //   to decrypt the data with,
        // - If using `SYMMETRIC` encoding, `key` must contain a 32 bytes symmetric key.
        extern __declspec(dllexport) char* gowaku_decode_data(char* data, char* keyType, char* key, int version);


byte[] data = Convert.FromBase64String(encodedString);
string decodedString = Encoding.UTF8.GetString(data);




    */
    }
}


