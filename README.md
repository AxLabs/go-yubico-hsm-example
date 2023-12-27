# Go example for YubiHSM2 FIPS

This repo laid the groundwork about how to set-up and interact with [YubiHSM2 FIPS](https://www.yubico.com/products/hardware-security-module/) using [Golang](https://go.dev/). This repository also intends to document architectural trade-offs and technical challenges.

 ## What the f*** is an HSM?

A Hardware Security Module (HSM) is a physical computing device that safeguards and manages digital keys for strong authentication and provides crypto-processing. HSM traditionally comes in the form of a plug-in or an external device that attaches directly to a computer or network server.

The main purpose of an HSM is to secure cryptographic keys and operations within the device, offering a higher level of security than software-based management because the keys are less susceptible to theft or unauthorized access.

👉 HSMs are specifically designed to protect the lifecycle of cryptographic keys. Their tamper-resistant physical design ensures that sensitive keys are never exposed outside the module.

👉 [FIPS](https://en.wikipedia.org/wiki/Federal_Information_Processing_Standards) (Federal Information Processing Standards) is developed by [NIST](https://www.nist.gov) (National Institute of Standards and Technology), a part of the [U.S. Department of Commerce](https://www.commerce.gov). FIPS standards are issued to establish requirements for various purposes such as ensuring computer security and interoperability.

👉 [YubiHSM2 FIPS](https://www.yubico.com/products/hardware-security-module/) product is certified with [FIPS 140-2, Level 3](https://en.wikipedia.org/wiki/FIPS_140-2).

## Install dependencies

First, you need to install the `yubihsm2-sdk`.

The SDK package, including all the tools, can be fetched from here: https://developers.yubico.com/YubiHSM2/Releases/

### MacOS

Install (`brew`):

```shell
brew install yubihsm2-sdk
```

If you don't know where the `yubihsm_pkcs11.dylib` is located, just do:

```shell
sudo find /usr -name "yubihsm_*.dylib" -print
```

### Linux (Ubuntu)

Go the [SDK release website](https://developers.yubico.com/YubiHSM2/Releases/), and select the package that is aligned to your Linux distro and version. For example, if you have Ubuntu 22.04, you will choose the file [`yubihsm2-sdk-2023-11-ubuntu2204-amd64.tar.gz`](https://developers.yubico.com/YubiHSM2/Releases/yubihsm2-sdk-2023-11-ubuntu2204-amd64.tar.gz).

After the download, extract it:

```shell
tar -xvzf yubihsm2-sdk-2023-11-ubuntu2204-amd64.tar.gz
cd yubihsm2-sdk
```

Install:

```
apt --fix-broken -y install $(ls ./*.deb | grep -v './libyubihsm-dev')
```

This command will install all `*.deb` files with the exception of the `libyubihsm-dev`, which is not strictly necessary.

You might need to add a `udev` rule. Create the file `/etc/udev/rules.d/` and add the following content:

```conf
# This udev file should be used with udev 188 and newer
ACTION!="add|change", GOTO="yubihsm2_connector_end"

# Yubico YubiHSM 2
# The OWNER attribute here has to match the uid of the process running the Connector
SUBSYSTEM=="usb", ATTRS{idVendor}=="1050", ATTRS{idProduct}=="0030", OWNER="yubihsm-connector"

LABEL="yubihsm2_connector_end"
```

## Architecture

In order to better understand the role of all the components and how they communicate, the following picture depicts a high-level architecture.

![Architecture Diagram](./docs/yubihsm-architecture.png)

Important points:

* The logical representation of nodes (i.e., machines) is just an example. All the components places within `Node N`, `Node C1`, and `Node C2`, can be placed in a single node/machine. The separation between 3 different nodes is just for understanding purposes -- mainly to highlight that `Client 2` doesn't require, e.g., `yubihsm-shell`, and that `Client 1` doesn't require, e.g., `yubihsm_pkcs11.so`, and so on.
* The native libraries (green), the connector (purple), and `Client 1` (blue), are distributed with the [YubiHSM2 SDK](https://developers.yubico.com/YubiHSM2/Releases/).
* An advantage of using [PKCS#11 standard](https://en.wikipedia.org/wiki/PKCS_11) to communicate with an HSM is interoperability. Therefore, it doesn't matter what exactly is behind the PKCS#11 interface, since it can be easily replaced without affecting the application implementation.

## YubiHSM Connector

From the developer portal at Yubi website:

> The Connector is not a trusted component. Sessions are established cryptographically between the application and the YubiHSM 2 using a symmetric mutual authentication scheme that is both encrypted and authenticated.

An important aspect about connectivity is also highlighted at the Yubico developer's website:

> The Connector is not required to run on the same host as the applications which access it. In that case the Connector should be configured to be listening on a different address and port rather than the default localhost:12345, making sure that the client has access.

Install:

```shell
yubihsm-connector install
```

Then, to start it:

```shell
sudo yubihsm-connector --config yubihsm-connector-config.yaml start
```

You might want to check if the `yubihsm-connector` is successfully running by using `curl`:

```shell
curl -v http://localhost:12345/connector/status
```

You should see something like this with a `HTTP 200` response:

```conf
status=OK
serial=*
version=3.0.4
pid=10276
address=localhost
port=12345
```

## Useful commands

* Start the `yubihsm-shell` in the interactive mode:

  ```shell
  yubihsm-shell
  ```

  Then you can connect and create a session:

  ```
  yubihsm> connect
  Session keepalive set up to run every 15 seconds
  yubihsm> session open 1 password
  Created session 0
  ```

  Then, you can, for example, list all objects:

  ```
  yubihsm> list objects 0
  Found 1 object(s)
  id: 0x0001, type: authentication-key, algo: aes128-yubico-authentication, sequence: 0, label: DEFAULT AUTHKEY CHANGE THIS ASAP
  ```

* You can also run the `yubihsm-shell` in a non-interactive mode, specifying what you want todo (e.g., actions) directly in the command line:

  ```shell
  yubihsm-shell --authkey=1 --password=password --outformat=hex --action=list-objects
  ```

## Scenario for key set-up

Let's imagine that you just bought the YubiHSM2 FIPS and would like to set-up the following:

- Generate an asymmetric key (secp256r1) that will never leave the HSM
- Enable this key to sign data

> 🚨**IMPORTANT**🚨: it's important to read [Core Concepts](https://docs.yubico.com/hardware/yubihsm-2/hsm-2-user-guide/hsm2-core-concepts.html) before you start. Make sure you understand what is an [Object](https://docs.yubico.com/hardware/yubihsm-2/hsm-2-user-guide/hsm2-core-concepts.html#object-id), [Capability](https://docs.yubico.com/hardware/yubihsm-2/hsm-2-user-guide/hsm2-core-concepts.html#capability) (including "Delegated Capabilities"), and [Domain](https://docs.yubico.com/hardware/yubihsm-2/hsm-2-user-guide/hsm2-core-concepts.html#domain).

### Setting up a new Authentication Key

Set-up a new auth key:

```shell
yubihsm-shell --authkey 1 --password password --outformat base64 -a put-authentication-key -i 0x0002 -l hsm-go-test -d 2 -c generate-asymmetric-key,export-wrapped,get-pseudo-random,put-wrap-key,import-wrapped,delete-asymmetric-key,sign-ecdsa --delegated sign-ecdsa,exportable-under-wrap,export-wrapped,import-wrapped --new-password newpassword123
```

Details:
* `-i 0x0002`: we're setting, by ourselves, the identifier of the object to `0x0002`. If you leave the `-i` param as  `0`, then a new one will be automatically assigned for you.
* `-d 2`: means that this authentication key is valid in the domain `2`.

> 🚨**IMPORTANT**🚨: the default authentication key (id: `0x0001`) should be deleted.

### Generate a key for signing & signning data

Generate the key (`ecdsa`, `secp256r1`):

```shell
yubihsm-shell --authkey 2 --password newpassword123 --outformat base64 -a generate-asymmetric-key -i 0 -l hsm-go-test-key1 -d 2 -c sign-ecdsa -A ecp256
```

Returns:

```
Generated Asymmetric key 0x13db
```

List all the objects (just to check if the key was created):

```shell
yubihsm-shell --authkey 2 --password newpassword123 --outformat base64 -a list-objects
```

Sign the data from the `data.txt` file (and output to `signature.b64`):

```shell
cat data.txt | yubihsm-shell --authkey 2 --password newpassword123 --outformat base64 -a sign-ecdsa -i 0x13db -A ecdsa-sha256
```

Get public key (and output to `asymmetric_key.pub`):

```shell
yubihsm-shell --authkey 2 --password newpassword123 --outformat base64 -a get-public-key -i 0x13db > asymmetric_key.pub
```

Convert the `signature.b64` to a binary format using `base64` cmd tool, and output to `signature.bin`:

  * If you're in MacOS:
  
    ```shell
    base64 -d -i signature.b64 > signature.bin
    ```

  * If you're in Linux:

    ```shell
    base64 -d signature.b64 > signature.bin
    ```

Verify the signature (from `signature.bin`) using `openssl`:

```shell
openssl dgst -sha256 -signature signature.bin -verify asymmetric_key.pub data.txt
```

## Signing using the Golang code

Make sure you take a look at the `.env.template` and the `const` section of the `main.go` file. If you would like to change any default value, the easiest way is to create a `.env` file and just override the varibales you need. All the others will get the default value.

The Golang example code uses [PKCS#11 standard](https://en.wikipedia.org/wiki/PKCS_11).

Run:

```shell
go run main.go
```

In summary, this is what `main.go` does:
- List all objects (keys)
- Fetch the private key object (i.e., object identifier)
- Test whether the private key value can be fetched (nooooo! 😅)
- Fetch the public key object (i.e., object identifier)
- Prints the public key (hex and base64)
- Requests a signature to the HSM (using `data.txt`)
- Prints the signature (hex and base64)
- Prints the curve based on the public key
- Verifies the signature based on the public key ✅ 🥳

> 🚨**IMPORTANT**🚨: DO NOT USE THIS CODE IN PRODUCTION. This is just an example and was done in the "quick and dirty" mode. 😅

## TODOs

- [ ] Clarify what's the best set-up in terms of auth key, audit key, and wrap key. Maybe a superior set-up is to have 3 keys with different roles.
- [ ] Clarify what's the best way to set-up the wrap key and backups.
- [ ] Docker image with all the tools from yubihsm SDK

## Acknowledgements and References

Some parts of this README were strongly based on the [user guide](https://docs.yubico.com/hardware/yubihsm-2/hsm-2-user-guide/hsm2-quick-start.html) provided by Yubico. They have well-written docs, no doubts. However, by re-writing some steps -- which are already part of the docs, but in a scattered manner -- made me better digest and explain details more clearly. I hope that's also the case for you. 😉

Also, thanks for [AxLabs](https://axlabs.com) to provide me the opportunity to play with an YubiHSM2 FIPS. 🙏🥳