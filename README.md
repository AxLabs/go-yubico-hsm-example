# Go example for YubiHSM2 FIPS

This repo laid the groundwork about how to set-up and interact with [YubiHSM2 FIPS](https://www.yubico.com/products/hardware-security-module/) using [Golang](https://go.dev/). This repository also intends to document architectural trade-offs and technical challenges.

 ## What the f*** is an HSM?

A Hardware Security Module (HSM) is a physical computing device that safeguards and manages digital keys for strong authentication and provides crypto-processing. HSM traditionally comes in the form of a plug-in or an external device that attaches directly to a computer or network server.

The main purpose of an HSM is to secure cryptographic keys and operations within the device, offering a higher level of security than software-based management because the keys are less susceptible to theft or unauthorized access.

👉 HSMs are specifically designed to protect the lifecycle of cryptographic keys. Their tamper-resistant physical design ensures that sensitive keys are never exposed outside the module.

👉 FIPS (Federal Information Processing Standards) is developed by NIST (National Institute of Standards and Technology), a part of the U.S. Department of Commerce. FIPS standards are issued to establish requirements for various purposes such as ensuring computer security and interoperability.

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
* An advantage of using PKCS#11 standard to communicate with an HSM is interoperability. Therefore, it doesn't matter what exactly is behind the PKCS#11 interface, since it can be easily replaced without affecting the application implementation.

## YubiHSM Connector

How to configure and stuff

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

## TODOs

- [ ] Suggest reading the following:
  - https://developers.yubico.com/YubiHSM2/Usage_Guides/YubiHSM_quick_start_tutorial.html
  - https://docs.yubico.com/hardware/yubihsm-2/hsm-2-user-guide/hsm2-quick-start.html
- [ ] Docker container to install yubihsm SDK and shell
- [ ] TBD.