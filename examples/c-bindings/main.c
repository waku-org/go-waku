#include <stdbool.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <time.h>
#include <stdint.h>
#include <inttypes.h>

#include "libgowaku.h"
#include "nxjson.c"
#include "main.h"

char *alicePrivKey = "0x4f012057e1a1458ce34189cb27daedbbe434f3df0825c1949475dec786e2c64e";
char *alicePubKey = "0x0440f05847c4c7166f57ae8ecaaf72d31bddcbca345e26713ca9e26c93fb8362ddcd5ae7f4533ee956428ad08a89cd18b234c2911a3b1c7fbd1c0047610d987302";

char *bobPrivKey = "0xb91d6b2df8fb6ef8b53b51b2b30a408c49d5e2b530502d58ac8f94e5c5de1453";
char *bobPubKey = "0x045eef61a98ba1cf44a2736fac91183ea2bd86e67de20fe4bff467a71249a8a0c05f795dd7f28ced7c15eaa69c89d4212cc4f526ca5e9a62e88008f506d850cccd";

void callBack(char *signal)
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

  const nx_json *json = nx_json_parse(signal, 0);
  const char *type = nx_json_get(json, "type")->text_value;

  if (strcmp(type, "message") == 0)
  {
    char* msg = utils_extract_wakumessage_from_signal(json);
    char *decodedMsg =  waku_decode_asymmetric(msg, bobPrivKey);
    free(msg);

    if(isError(decodedMsg)) {
      free(decodedMsg);
      return;
    }


    const nx_json *dataJson = nx_json_parse(decodedMsg, 0);
    const char *pubkey = nx_json_get(nx_json_get(dataJson, "result"), "pubkey")->text_value;
    const char *base64data = nx_json_get(nx_json_get(dataJson, "result"), "data")->text_value;
    char *data =  waku_utils_base64_decode((char*)base64data);

    printf(">>> Received \"%s\" from %s\n", utils_get_str(data), pubkey);
    fflush(stdout);
    free(data);
  }

  nx_json_free(json);
}

int main(int argc, char *argv[])
{
  char *response;
  waku_set_event_callback(callBack);

  char *configJSON = "{\"host\": \"0.0.0.0\", \"port\": 60000, \"logLevel\":\"error\", \"store\":true}";
  response = waku_new(configJSON); // configJSON can be NULL too to use defaults
  if (isError(response))
    return 1;

  response = waku_start(); // Start the node, enabling the waku protocols
  if (isError(response))
    return 1;

  response = waku_peerid(); // Obtain the node peerID
  if (isError(response))
    return 1;
  char *nodePeerID = utils_get_str(response);
  printf("PeerID: %s\n", nodePeerID);

  
  response =  waku_connect("/dns4/node-01.gc-us-central1-a.wakuv2.test.statusim.net/tcp/30303/p2p/16Uiu2HAmJb2e28qLXxT5kZxVUUoJt72EMzNGXB47Rxx5hw3q4YjS", 0); // Connect to a node
  if (isError(response))
    printf("Could not connect to node: %s\n", response);


  /*
  // To use dns discovery, and retrieve nodes from a enrtree url
  response = waku_dns_discovery("enrtree://AOGECG2SPND25EEFMAJ5WF3KSGJNSGV356DSTL2YVLLZWIV6SAYBM@test.waku.nodes.status.im", "", 0); // Discover Nodes
  if (isError(response))
    return 1;
  printf("Discovered nodes: %s\n", response);
  */
  
  /*
  // To see a store query in action:
  char query[1000];
  sprintf(query, "{\"pubsubTopic\":\"%s\", \"pagingOptions\":{\"pageSize\": 40, \"forward\":false}}", waku_default_pubsub_topic());
  response = waku_store_query(query, NULL, 0);
  if (isError(response))
    return 1;
  printf("%s\n",response);
  */
  response = waku_relay_subscribe(NULL);
  if (isError(response))
    return 1;

  int i = 0;
  int version = 1;
  while (i < 5)
  {
    i++;

    char wakuMsg[1000];
    char *msgPayload = waku_utils_base64_encode("Hello World!");
    char *contentTopic = waku_content_topic("example", 1, "default", "rfc26");
    sprintf(wakuMsg, "{\"payload\":\"%s\",\"contentTopic\":\"%s\",\"timestamp\":%"PRIu64"}", msgPayload, contentTopic, nowInNanosecs());
    free(msgPayload);
    free(contentTopic);


    response = waku_relay_publish_enc_asymmetric(wakuMsg, NULL, bobPubKey, alicePrivKey, 0); // Broadcast via waku relay a message encrypting it with Bob's PubK, and signing it with Alice PrivK
    // response = waku_lightpush_publish_enc_asymmetric(wakuMsg, NULL, NULL, bobPubKey, alicePrivKey, 0); // Broadcast via waku lightpush a message encrypting it with Bob's PubK, and signing it with Alice PrivK
    if (isError(response))
      return 1;
    
    // char *messageID = utils_get_str(response);
    // free(messageID);

    sleep(1);
  }

    
  // To retrieve messages from local store, set store:true in the node config, and use waku_store_local_query
  /*
  char query[1000];
  sprintf(query, "{\"pubsubTopic\":\"%s\", \"pagingOptions\":{\"pageSize\": 40, \"forward\":false}}", waku_default_pubsub_topic());
  response = waku_store_local_query(query);
  if (isError(response))
    return 1;
  printf("%s\n",response);
  */

  response = waku_stop();
  if (isError(response))
    return 1;

  return 0;
}
