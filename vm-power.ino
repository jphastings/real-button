#include "DigiKeyboard.h"

#define MODIFIER 125
#define KEY 88

#define BUTTON 0
#define LED 1

void setup() {
  pinMode(LED, OUTPUT);
  pinMode(BUTTON, INPUT_PULLUP);
}

void loop() {
  digitalWrite(LED, LOW);
  if (!digitalRead(BUTTON)) {
    digitalWrite(LED, HIGH);
    DigiKeyboard.sendKeyStroke(KEY, MODIFIER);
    DigiKeyboard.delay(1000);
  }
  DigiKeyboard.delay(5);
}
