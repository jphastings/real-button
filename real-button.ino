struct RGBButton {
  int buttonPin;
  int rPin;
  int gPin;
  int bPin;
};

// Add a line here for each RGB button attached. No more than 30 (even if you have a very large number of IO pins!)
const RGBButton button_pins[] = {
  {12,11,10,9}
};

#define debounce 100 // ms
#define poll 10 // ms
const int BUTTON_COUNT = sizeof(button_pins) / sizeof(button_pins[0]);
int latches[BUTTON_COUNT];

void setup() {
  pinMode(LED_BUILTIN, OUTPUT);
  digitalWrite(LED_BUILTIN, HIGH);
  Serial.begin(9600);
  for (int pin = 0; pin < BUTTON_COUNT; pin++) {
    pinMode(button_pins[pin].buttonPin, INPUT_PULLUP);
    pinMode(button_pins[pin].rPin, OUTPUT);
    pinMode(button_pins[pin].gPin, OUTPUT);
    pinMode(button_pins[pin].bPin, OUTPUT);
  }
  digitalWrite(LED_BUILTIN, LOW);
}

void loop() {
  bool allOff = true;
  for (int pin = 0; pin < BUTTON_COUNT; pin++) {
    allOff = !checkButton(pin) && allOff;
  }
  if (allOff) {
      digitalWrite(LED_BUILTIN, LOW);
  }
  checkLEDs();
  delay(poll);
}

// checkButton sees whether a button is pressed, sends a message over serial if it is
// and returns "true" immediately afterwards, and continuously until the debounce period
// is completed.
bool checkButton(int pin) {
  if (!digitalRead(button_pins[pin].buttonPin)) {
    if(!latches[pin]) {
      digitalWrite(LED_BUILTIN, HIGH);
      Serial.write(pin);
      latches[pin] = debounce;
    }
  } else {
    if (latches[pin] > 0) {
      latches[pin] -= poll;
    }
  }

  return latches[pin] > 0;
}

void checkLEDs() {
  if (Serial.available() == 0) {
    return;
  }

  int command = Serial.read();
  int pin = command & 0b00011111;
  bool r = (command & 0b10000000) > 0;
  bool g = (command & 0b01000000) > 0;
  bool b = (command & 0b00100000) > 0;

  // Special Commands
  if (pin == 31) {
    specialResponses(command >> 5);
    return;
  }

  if (pin >= BUTTON_COUNT) {
    return;
  }

  digitalWrite(button_pins[pin].rPin, r);
  digitalWrite(button_pins[pin].gPin, g);
  digitalWrite(button_pins[pin].bPin, b);
}

void specialResponses(int cmd) {
  Serial.write(0b11100000 + BUTTON_COUNT);
}
