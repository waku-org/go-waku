#ifndef MAIN_H
#define MAIN_H

#include <stdbool.h>
#include "nxjson.c"

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


char *utils_extract_wakumessage_from_signal(const nx_json *signal)
{
    const nx_json *wakuMsgJson = nx_json_get(nx_json_get(signal, "event"), "wakuMessage");
    const char *payload = nx_json_get(wakuMsgJson, "payload")->text_value;
    const char *contentTopic = nx_json_get(wakuMsgJson, "contentTopic")->text_value;
    int version = nx_json_get(wakuMsgJson, "version")->int_value;
    long long timestamp = nx_json_get(wakuMsgJson, "timestamp")->int_value;
    char wakuMsg[1000];
    sprintf(wakuMsg, "{\"payload\":\"%s\",\"contentTopic\":\"%s\",\"timestamp\":%lld, \"version\":%d}", payload, contentTopic, timestamp, version);
    char *response = (char *)malloc(sizeof(char) * (strlen(wakuMsg) + 1));
    strcpy(response, wakuMsg);
    return response;
}

long long utils_get_int(char *input)
{
    char *jsonStr = malloc(strlen(input) + 1);
    strcpy(jsonStr, input);
    const nx_json *json = nx_json_parse(jsonStr, 0);
    long long result = -1;
    if (json)
    {
        result = nx_json_get(json, "result")->int_value;
    }
    nx_json_free(json);
    free(jsonStr);

    return result;
}

char *utils_get_str(char *input)
{
    char *jsonStr = malloc(strlen(input) + 1);
    strcpy(jsonStr, input);
    const nx_json *json = nx_json_parse(jsonStr, 0);
    char *result = "";
    if (json)
    {
        const char *text_value = nx_json_get(json, "result")->text_value;
        result = strdup(text_value);
    }

    nx_json_free(json);
    free(jsonStr);

    return result;
}

#endif /* MAIN_H */