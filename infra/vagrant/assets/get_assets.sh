#!/usr/bin/env bash
#First recording
curl \
  -X POST \
  -H "Authorization: Token $YOUR_TOKEN" \
  -H "Content-Type: text/plain" \
  -d "Hello, and welcome.
You are connected to FRED, your virtual assistant.

To record your message, please press 1.
When you finish recording, press the pound key.

To end the call at any time, press 0." \
  "https://api.deepgram.com/v1/speak?model=aura-2-jupiter-en&encoding=linear16&container=wav&sample_rate=8000" \
  -o welcome-ari.wav

#Second recording
curl \
  -X POST \
  -H "Authorization: Token $YOUR_TOKEN" \
  -H "Content-Type: text/plain" \
  -d "Your message has been recorded.

To record your message again, press 1.
To listen to your recording, press 2.
To confirm and submit your request, press 3.

Press the pound key to stop listening or recording.
To end the call, press 0. " \
  "https://api.deepgram.com/v1/speak?model=aura-2-jupiter-en&encoding=linear16&container=wav&sample_rate=8000" \
  -o after-recording.wav

#After response
curl \
  -X POST \
  -H "Authorization: Token $YOUR_TOKEN" \
  -H "Content-Type: text/plain" \
  -d "Your request has been processed.

To record your message again, press 1.
To listen to your recording, press 2.
To confirm and submit your request, press 3.
To listen to the response to your request, press 4.

Press the pound key to stop listening or playback.
To end the call, press 0."
"https://api.deepgram.com/v1/speak?model=aura-2-jupiter-en&encoding=linear16&container=wav&sample_rate=8000" \
  -o after-responding.wav

#Good bye
curl \
  -X POST \
  -H "Authorization: Token $YOUR_TOKEN" \
  -H "Content-Type: text/plain" \
  -d "Thank you for calling FRED.
Goodbye, and have a nice day." \
  "https://api.deepgram.com/v1/speak?model=aura-2-jupiter-en&encoding=linear16&container=wav&sample_rate=8000" \
  -o ari_goodbye
