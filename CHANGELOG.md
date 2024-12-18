# Changelog

## [0.0.4](https://github.com/googleapis/genai-toolbox/compare/v0.0.3...v0.0.4) (2024-12-18)


### Features

* Add `auth_required` to tools ([#123](https://github.com/googleapis/genai-toolbox/issues/123)) ([3118104](https://github.com/googleapis/genai-toolbox/commit/3118104ae17335db073911a88f2ea8ce8d0bfb45))
* Add Auth Source configuration ([#71](https://github.com/googleapis/genai-toolbox/issues/71)) ([77b0d43](https://github.com/googleapis/genai-toolbox/commit/77b0d4317580214c1c9bd542b24371f09fd17fe0))
* Add Tool authenticated parameters ([#80](https://github.com/googleapis/genai-toolbox/issues/80)) ([380a6fb](https://github.com/googleapis/genai-toolbox/commit/380a6fbbd5a5abc3159c96421b0923c117807267))
* **langchain-sdk:** Correctly parse Manifest API response as JSON ([#143](https://github.com/googleapis/genai-toolbox/issues/143)) ([2c8633c](https://github.com/googleapis/genai-toolbox/commit/2c8633c3eb2d936b62fe24c87a6385d5898f4370))
* **langchain-sdk:** Support authentication in LangChain Toolbox SDK. ([#133](https://github.com/googleapis/genai-toolbox/issues/133)) ([23fa912](https://github.com/googleapis/genai-toolbox/commit/23fa912a80e7e02f53a5ad27781e32a5cfa05458))


### Bug Fixes

* Fix release image version tag ([#136](https://github.com/googleapis/genai-toolbox/issues/136)) ([6d19ff9](https://github.com/googleapis/genai-toolbox/commit/6d19ff96e4004c97739ad6a064ef72e57f8da2f2))
* **langchain-sdk:** Correct test name to ensure execution and full coverage. ([#145](https://github.com/googleapis/genai-toolbox/issues/145)) ([d820ac3](https://github.com/googleapis/genai-toolbox/commit/d820ac3767127058dc726b44e469a7adec26783b))
* Set server version ([#150](https://github.com/googleapis/genai-toolbox/issues/150)) ([abd1eb7](https://github.com/googleapis/genai-toolbox/commit/abd1eb702c1ab75d76be624d2f0decd34548f93f))


### Miscellaneous Chores

* Release 0.0.4 ([#152](https://github.com/googleapis/genai-toolbox/issues/152)) ([86ec12f](https://github.com/googleapis/genai-toolbox/commit/86ec12f8c5d67ced5bcd52c9d8e80b17aa11b514))

## [0.0.3](https://github.com/googleapis/genai-toolbox/compare/v0.0.2...v0.0.3) (2024-12-10)


### Features

* Add --log-level and --logging-format flags ([#97](https://github.com/googleapis/genai-toolbox/issues/97)) ([9a0f618](https://github.com/googleapis/genai-toolbox/commit/9a0f618efca13e0accb2656ea74a393e8cda5d40))
* Add options for command ([#110](https://github.com/googleapis/genai-toolbox/issues/110)) ([5c690c5](https://github.com/googleapis/genai-toolbox/commit/5c690c5c30515ae790b045677ef518106c52a491))
* Add Spanner source and tool ([#90](https://github.com/googleapis/genai-toolbox/issues/90)) ([890914a](https://github.com/googleapis/genai-toolbox/commit/890914aae0989d181b26efa940326a5c2f559959))
* Add std logger ([#95](https://github.com/googleapis/genai-toolbox/issues/95)) ([6a8feb5](https://github.com/googleapis/genai-toolbox/commit/6a8feb51f0d148607f52c4a5c755faa9e3b7e6a4))
* Add structured logger ([#96](https://github.com/googleapis/genai-toolbox/issues/96)) ([5e20417](https://github.com/googleapis/genai-toolbox/commit/5e2041755163932c6c3135fad2404cffd22cb463))
* **source/alloydb-pg:** Add configuration for public and private IP ([#103](https://github.com/googleapis/genai-toolbox/issues/103)) ([e88ec40](https://github.com/googleapis/genai-toolbox/commit/e88ec409d14c85d6b0896c45d9957cce9097912a))
* **source/cloudsql-pg:** Add configuration for public and private IP ([#114](https://github.com/googleapis/genai-toolbox/issues/114)) ([6479c1d](https://github.com/googleapis/genai-toolbox/commit/6479c1dbe26f05438df9c2289118da558eee0a0d))


### Bug Fixes

* Fix go test workflow ([#84](https://github.com/googleapis/genai-toolbox/issues/84)) ([8c2c373](https://github.com/googleapis/genai-toolbox/commit/8c2c373d359b718b2182f566bc245a2a8fa03333))
* Fix issue causing client session to not close properly while closing SDK. ([#81](https://github.com/googleapis/genai-toolbox/issues/81)) ([9d360e1](https://github.com/googleapis/genai-toolbox/commit/9d360e16eab664992bca9d6b01dbec12c9d5d2e1))
* Fix test cases for ip_type ([#115](https://github.com/googleapis/genai-toolbox/issues/115)) ([5528bec](https://github.com/googleapis/genai-toolbox/commit/5528bec8ed8c7efa03979abedc98102bff4abed8))
* Fix the errors showing up after setting up mypy type checker. ([#74](https://github.com/googleapis/genai-toolbox/issues/74)) ([522bbef](https://github.com/googleapis/genai-toolbox/commit/522bbefa7b305a1695bb21ce4a9c92429cde4ee9))
* **llamaindex-sdk:** Fix issue causing client session to not close properly while closing SDK. ([#82](https://github.com/googleapis/genai-toolbox/issues/82)) ([fa03376](https://github.com/googleapis/genai-toolbox/commit/fa03376bbc4b9dba93a471b13225c8f1a37187c2))


### Miscellaneous Chores

* Release 0.0.3 ([#122](https://github.com/googleapis/genai-toolbox/issues/122)) ([626e12f](https://github.com/googleapis/genai-toolbox/commit/626e12fdb3e27996e9e4a8c9661563ec3c3bcc5c))

## [0.0.2](https://github.com/googleapis/genai-toolbox/compare/v0.0.1...v0.0.2) (2024-11-12)


### ⚠ BREAKING CHANGES

* consolidate "x-postgres-generic" tools to "postgres-sql" tool ([#43](https://github.com/googleapis/genai-toolbox/issues/43))

### Features

* Consolidate "x-postgres-generic" tools to "postgres-sql" tool ([#43](https://github.com/googleapis/genai-toolbox/issues/43)) ([f630965](https://github.com/googleapis/genai-toolbox/commit/f6309659374bc9cb500cc54dd4220baa0a451a3b))
* **container:** Add entrypoint in Dockerfile ([#38](https://github.com/googleapis/genai-toolbox/issues/38)) ([b08072a](https://github.com/googleapis/genai-toolbox/commit/b08072a80034a34a394dea82838422bd6cb0d23a))
* **sdk:** Added LlamaIndex SDK ([#48](https://github.com/googleapis/genai-toolbox/issues/48)) ([b824abe](https://github.com/googleapis/genai-toolbox/commit/b824abe72fbf518ec91fb12e5270c0a19e776d2f))
* **sdk:** Make ClientSession optional when initializing ToolboxClient ([#55](https://github.com/googleapis/genai-toolbox/issues/55)) ([26347b5](https://github.com/googleapis/genai-toolbox/commit/26347b5a5e71434d7bd2b7a9e6458247e75e3969))
* Support requesting a single tool ([#56](https://github.com/googleapis/genai-toolbox/issues/56)) ([efafba9](https://github.com/googleapis/genai-toolbox/commit/efafba9033e046905552f149f59893a4fad41afb))


### Bug Fixes

* Correct source type validation for postgres-sql tool ([#47](https://github.com/googleapis/genai-toolbox/issues/47)) ([52ebb43](https://github.com/googleapis/genai-toolbox/commit/52ebb431b784d160508273492d904d3b101afeb9))
* **docs:** Correct outdated references to tool kinds ([#49](https://github.com/googleapis/genai-toolbox/issues/49)) ([972888b](https://github.com/googleapis/genai-toolbox/commit/972888b9d64e1fea1d9a56b13268235ea55b9d66))
* Handle content-type correctly ([#33](https://github.com/googleapis/genai-toolbox/issues/33)) ([cf8112f](https://github.com/googleapis/genai-toolbox/commit/cf8112f85610833f2f4f2817a65fc4f7cf2322d8))


### Miscellaneous Chores

* Release 0.0.2 ([#65](https://github.com/googleapis/genai-toolbox/issues/65)) ([beea3c3](https://github.com/googleapis/genai-toolbox/commit/beea3c32d94d605973ba06b71a37b7c1bd4787bf))

## 0.0.1 (2024-10-28)


### Features

* Add address and port flags ([#7](https://github.com/googleapis/genai-toolbox/issues/7)) ([df9ad9e](https://github.com/googleapis/genai-toolbox/commit/df9ad9e33f99e6e5b692d9a99c2a90fbe3667265))
* Add AlloyDB source and tool ([#23](https://github.com/googleapis/genai-toolbox/issues/23)) ([fe92d02](https://github.com/googleapis/genai-toolbox/commit/fe92d02ae2ac2e70769dd2ee177cab91233a01cd))
* Add basic CLI ([#5](https://github.com/googleapis/genai-toolbox/issues/5)) ([1539ee5](https://github.com/googleapis/genai-toolbox/commit/1539ee56dddbee3a19069ef887375e76503fbdbd))
* Add basic http server ([#6](https://github.com/googleapis/genai-toolbox/issues/6)) ([e09ae30](https://github.com/googleapis/genai-toolbox/commit/e09ae30a90083a3777f91dd661e5a85bacdd48ba))
* Add basic parsing from tools file ([#8](https://github.com/googleapis/genai-toolbox/issues/8)) ([b9ba364](https://github.com/googleapis/genai-toolbox/commit/b9ba364fb66a884178d207e57310e07cf8d6cff1))
* Add initial cloud sql pg invocation ([#14](https://github.com/googleapis/genai-toolbox/issues/14)) ([3703176](https://github.com/googleapis/genai-toolbox/commit/3703176fce110ebb999deeb73d6b3aba29dee276))
* Add Postgres source and tool ([#25](https://github.com/googleapis/genai-toolbox/issues/25)) ([2742ed4](https://github.com/googleapis/genai-toolbox/commit/2742ed48b8d52f748a9edbc520068e1b88d82758))
* Add preliminary parsing of parameters ([#13](https://github.com/googleapis/genai-toolbox/issues/13)) ([27edd3b](https://github.com/googleapis/genai-toolbox/commit/27edd3b5f671b2ce7677729fae4e56381271c990))
* Add support for array type parameters ([#26](https://github.com/googleapis/genai-toolbox/issues/26)) ([3903e86](https://github.com/googleapis/genai-toolbox/commit/3903e860bc67a7b385e316220ba4ea37e00c20f2))
* Add toolset configuration ([#12](https://github.com/googleapis/genai-toolbox/issues/12)) ([59b4bc0](https://github.com/googleapis/genai-toolbox/commit/59b4bc07f4b8521c188d10ed047eee817d19e424))
* Add Toolset manifest endpoint ([#11](https://github.com/googleapis/genai-toolbox/issues/11)) ([61e7b78](https://github.com/googleapis/genai-toolbox/commit/61e7b78ad8af2e51f824ced32d14234fa32da30a))
* **langchain-sdk:** Add Toolbox SDK for LangChain ([#22](https://github.com/googleapis/genai-toolbox/issues/22)) ([0bcd4b6](https://github.com/googleapis/genai-toolbox/commit/0bcd4b6e418a8e43f2b7b74a0969da171e2081bf))
* Stub basic control plane functionality  ([#9](https://github.com/googleapis/genai-toolbox/issues/9)) ([336bdc4](https://github.com/googleapis/genai-toolbox/commit/336bdc4d56580637afff2313bef64b50b148faca))


### Miscellaneous Chores

* Release 0.0.1 ([#31](https://github.com/googleapis/genai-toolbox/issues/31)) ([1f24ddd](https://github.com/googleapis/genai-toolbox/commit/1f24dddb4b24ff4336998bf43acaf4607a48ff66))


### Continuous Integration

* Add realease-please ([#15](https://github.com/googleapis/genai-toolbox/issues/15)) ([17fbbb4](https://github.com/googleapis/genai-toolbox/commit/17fbbb49b05996c2c43df4b72cf08488224c522a))
