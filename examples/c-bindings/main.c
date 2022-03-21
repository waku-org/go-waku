#include <stdbool.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>

#include "libgowaku.h"
#include "nxjson.c"
#include "main.h"


char *alicePrivKey = "0x4f012057e1a1458ce34189cb27daedbbe434f3df0825c1949475dec786e2c64e";
char *alicePubKey = "0x0440f05847c4c7166f57ae8ecaaf72d31bddcbca345e26713ca9e26c93fb8362ddcd5ae7f4533ee956428ad08a89cd18b234c2911a3b1c7fbd1c0047610d987302";


char *bobPrivKey = "0xb91d6b2df8fb6ef8b53b51b2b30a408c49d5e2b530502d58ac8f94e5c5de1453";
char *bobPubKey = "0x045eef61a98ba1cf44a2736fac91183ea2bd86e67de20fe4bff467a71249a8a0c05f795dd7f28ced7c15eaa69c89d4212cc4f526ca5e9a62e88008f506d850cccd";


int main(int argc, char *argv[])
{
  char *response;
  gowaku_set_event_callback(callBack);

  char *configJSON = "{\"host\": \"0.0.0.0\", \"port\": 60000}";
  response = gowaku_new(configJSON); // configJSON can be NULL too to use defaults
  if (isError(response))
    return 1;
  int nodeID = getIntValue(response); // Obtain the nodeID from the response



  response = gowaku_start(nodeID); // Start the node, enabling the waku protocols
  if (isError(response))
    return 1;



  response = gowaku_id(nodeID); // Obtain the node peerID
  if (isError(response))
    return 1;
  char *nodePeerID = getStrValue(response);
  printf("PeerID: %s\n", nodePeerID);


  /*
  response = gowaku_dial_peer(nodeID, "/dns4/node-01.gc-us-central1-a.wakuv2.test.statusim.net/tcp/30303/p2p/16Uiu2HAmJb2e28qLXxT5kZxVUUoJt72EMzNGXB47Rxx5hw3q4YjS", 0); // Connect to a node
  if (isError(response))
    return 1;
  */


  response = gowaku_relay_subscribe(nodeID, NULL);
  if (isError(response))
    return 1;
  char *subscriptionID = getStrValue(response);
  printf("SubscriptionID: %s\n", subscriptionID);



  int i = 0;
  int version = 1;
  while (true){
      i++;
      
      response = gowaku_encode_data("Hello World!", ASYMMETRIC, bobPubKey, alicePrivKey, version); // Send a message encrypting it with Bob's PubK, and signing it with Alice PrivK
      if (isError(response))
        return 1;
      char *encodedData = getStrValue(response);


      char *contentTopic = getStrValue(gowaku_content_topic("example", 1, "default", "rfc26"));


      char wakuMsg[1000];
      sprintf(wakuMsg, "{\"payload\":\"%s\",\"contentTopic\":\"%s\",\"version\":%d,\"timestamp\":%d}", encodedData, contentTopic, version, i);

      response = gowaku_relay_publish(nodeID, wakuMsg, NULL, 0); // Broadcast a message
      if (isError(response))
        return 1;
      // char *messageID = getStrValue(response);

      sleep(1);
  }



  response = gowaku_stop(nodeID);
  if (isError(response))
    return 1;

  return 0;
}

void callBack(char *signal)
{
  // This callback will be executed each time a new message is received

  // Example signal:
  /*{
      "nodeId":1,
      "type":"message",
      "event":{
        "messageID":"0x6496491e40dbe0b6c3a2198c2426b16301688a2daebc4f57ad7706115eac3ad1",
        "pubsubTopic":"/waku/2/default-waku/proto",
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

  if (strcmp(type,"message") == 0){
    const char *encodedPayload = nx_json_get(nx_json_get(nx_json_get(json, "event"), "wakuMessage"), "payload")->text_value;
    int version = nx_json_get(nx_json_get(nx_json_get(json, "event"), "wakuMessage"), "version")->int_value;
    
    char *decodedData = gowaku_decode_data((char*)encodedPayload, ASYMMETRIC, bobPrivKey, version);
    if(isError(decodedData)) return;

    const nx_json *dataJson = nx_json_parse(decodedData, 0);
    const char *pubkey = nx_json_get(nx_json_get(dataJson, "result"), "pubkey")->text_value;
    const char *base64data = nx_json_get(nx_json_get(dataJson, "result"), "data")->text_value;
    char *data = gowaku_utils_base64_decode((char*)base64data);

    printf("Received \"%s\" from %s\n", getStrValue(data), pubkey);
    fflush(stdout);

    nx_json_free(dataJson);    
  }

  nx_json_free(json);
}

bool isError(char *input)
{
  char *jsonStr = malloc(strlen(input) + 1);
  strcpy(jsonStr, input);
  const nx_json *json = nx_json_parse(jsonStr, 0);
  bool result = false;
  if (json)
  {
    const char *errTxt = nx_json_get(json, "error")->text_value;
    result = errTxt != NULL;
    if (result)
    {
      printf("ERROR: %s\n", errTxt);
    }
  }
  nx_json_free(json);
  free(jsonStr);
  return result;
}

int getIntValue(char *input)
{
  char *jsonStr = malloc(strlen(input) + 1);
  strcpy(jsonStr, input);
  const nx_json *json = nx_json_parse(jsonStr, 0);
  int result = -1;
  if (json)
  {
    result = nx_json_get(json, "result")->int_value;
  }
  nx_json_free(json);
  free(jsonStr);

  return result;
}

char* getStrValue(char *input)
{
  char *jsonStr = malloc(strlen(input) + 1);
  strcpy(jsonStr, input);
  const nx_json *json = nx_json_parse(jsonStr, 0);
  char* result = "";
  if (json)
  {
    const char* text_value = nx_json_get(json, "result")->text_value;
    result = strdup(text_value);
  }

  nx_json_free(json);
  free(jsonStr);

  return result;
}
