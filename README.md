# Aisc Aisc

[![Go](https://github.com/aisc/workflows/Go/badge.svg)](https://github.com/aisc/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/aisc.svg)](https://pkg.go.dev/github.com/aisc)
[![codecov](https://codecov.io/gh/ethersphere/aisc/branch/master/graph/badge.svg?token=63RNRLO3RU)](https://codecov.io/gh/ethersphere/aisc)
[![Go Report Card](https://goreportcard.com/badge/github.com/aisc)](https://goreportcard.com/report/github.com/aisc)
[![API OpenAPI Specs](https://img.shields.io/badge/openapi-api-blue)](https://docs.aisc.org/api/)
[![Debug API OpenAPI Specs](https://img.shields.io/badge/openapi-debugapi-lightblue)](https://docs.aisc.org/debug-api/)
![Docker Pulls](https://img.shields.io/docker/pulls/ethersphere/aisc)
![GitHub all releases](https://img.shields.io/github/downloads/ethersphere/aisc/total)
![GitHub](https://img.shields.io/github/license/ethersphere/aisc)


## DISCLAIMER
This software is provided to you "as is", use at your own risk and without warranties of any kind.
It is your responsibility to read and understand how Aisc works and the implications of running this software.
The usage of Aisc involves various risks, including, but not limited to:
damage to hardware or loss of funds associated with the Ethereum account connected to your node.
No developers or entity involved will be liable for any claims and damages associated with your use,
inability to use, or your interaction with other nodes or the software.

Our documentation is hosted at https://docs.aisc.org.

## Versioning

There are two versioning schemes used in Aisc that you should be aware of. The main Aisc version does **NOT** follow
strict Semantic Versioning. Aisc hosts different peer-to-peer wire protocol implementations and individual protocol breaking changes would necessitate a bump in the major part of the version. Breaking changes are expected with bumps of the minor version component. New (backward-compatible) features and bug fixes are expected with a bump of the patch component. Major version bumps are reserved for significant changes in Aisc's incentive structure.


The second set of versions that are important are the Aisc's API versions (denoted in our [Aisc](https://github.com/aisc/blob/master/openapi/Aisc.yaml) and [Aisc Debug](https://github.com/aisc/blob/master/openapi/AiscDebug.yaml) OpenAPI specifications). These versions **do follow**
Semantic Versioning and hence you should follow these for breaking changes.

## Contributing

Please read the [coding guidelines](CODING.md) and [style guide](CODINGSTYLE.md).

## Installing

[Install instructions](https://docs.aisc.org/docs/installation/quick-start)

## Get in touch
[Only official website](https://www. aisc.org)


## License

This library is distributed under the BSD-style license found in the [LICENSE](LICENSE) file.
