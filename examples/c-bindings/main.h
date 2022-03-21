#ifndef MAIN_H
#define MAIN_H

#include <stdbool.h>

void callBack(char *signal);

bool isError(char *input);

int getIntValue(char *input);

char* getStrValue(char *input);

#endif  /* MAIN_H */