# Using the `noise` application

## Background

The `noise` application is an example that shows how to do pairing between js-waku and go-waku

## Preparation
```
make
```

Also clone https://github.com/waku-org/js-noise and execute the following commands:
```
git clone https://github.com/waku-org/js-noise
cd js-noise
npm install
npm run build
cd example
npm install
npm start
```

## Basic application usage

To start the `noise` application run the following from the project directory

```
./build/noise
```
The app will display a QR and a link that will open the js-noise example in the browser. The handshake process will start afterwards
