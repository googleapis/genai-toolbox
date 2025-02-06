# Changelog

## [0.1.0](https://github.com/googleapis/genai-toolbox/compare/v0.0.5...v0.1.0) (2025-02-06)


### ⚠ BREAKING CHANGES

* **langchain-sdk:** Migrate existing state and APIs to a tools level class. ([#171](https://github.com/googleapis/genai-toolbox/issues/171))

### Features

* Add Cloud SQL for SQL Server Source and Tool ([#223](https://github.com/googleapis/genai-toolbox/issues/223)) ([9bad952](https://github.com/googleapis/genai-toolbox/commit/9bad9520604aa363a6d73f5ce14686895c2f4333))
* Add cloudsql mysql source and tool ([#221](https://github.com/googleapis/genai-toolbox/issues/221)) ([f1f61d7](https://github.com/googleapis/genai-toolbox/commit/f1f61d70877a1c7cc9080f6d70112bd0c0533473))
* Add dgraph tool and source ([#233](https://github.com/googleapis/genai-toolbox/issues/233)) ([617cc87](https://github.com/googleapis/genai-toolbox/commit/617cc872d1d692138a712d39fb7c1a405e9c1876))
* Add local quickstart ([#232](https://github.com/googleapis/genai-toolbox/issues/232)) ([497fb06](https://github.com/googleapis/genai-toolbox/commit/497fb06fae6d04adaad11fa78eb04282d0225dbd))
* Add user agent to cloud databases ([#244](https://github.com/googleapis/genai-toolbox/issues/244)) ([8452f8e](https://github.com/googleapis/genai-toolbox/commit/8452f8eb4457dcb0e360a9d9ae5b6e14e78806b1))
* **langchain-sdk:** Add features for binding parameters to ToolboxTool class. ([#192](https://github.com/googleapis/genai-toolbox/issues/192)) ([e0ee128](https://github.com/googleapis/genai-toolbox/commit/e0ee1286f924c87e9adf66a90ecb5eb0371b154d))
* **langchain-sdk:** Migrate existing state and APIs to a tools level class. ([#171](https://github.com/googleapis/genai-toolbox/issues/171)) ([a136308](https://github.com/googleapis/genai-toolbox/commit/a136308f5508a4a2dcc4f8ee672eed961cdfcb64))
* **mysql:** Add source ([#250](https://github.com/googleapis/genai-toolbox/issues/250)) ([378692a](https://github.com/googleapis/genai-toolbox/commit/378692ab50a90dcc1c3353052d0741cfd318c79d))
* **sqlserver:** Add mssql source ([#255](https://github.com/googleapis/genai-toolbox/issues/255)) ([8fca0a9](https://github.com/googleapis/genai-toolbox/commit/8fca0a95ee5e79e30919b05592af643ba57f3183))


### Bug Fixes

* Add coverage file to .gitignore ([#228](https://github.com/googleapis/genai-toolbox/issues/228)) ([e2faaf2](https://github.com/googleapis/genai-toolbox/commit/e2faaf2069b370dc1762193896f4a5962234594b))
* Auth token verification failure should not throw error immedicately ([#234](https://github.com/googleapis/genai-toolbox/issues/234)) ([4639cc6](https://github.com/googleapis/genai-toolbox/commit/4639cc6560f09b6b8203650ccce424ce59aa0c14))
* Fix typo in postgres test ([#216](https://github.com/googleapis/genai-toolbox/issues/216)) ([0c3d12a](https://github.com/googleapis/genai-toolbox/commit/0c3d12ae04a752fddcff06e92967910cdd643bbf))
* **langchain-sdk:** Inherit from BaseTool and implement placeholder _run method. ([779bf6b](https://github.com/googleapis/genai-toolbox/commit/779bf6b9a28ccb2f788ca1c99e8bdf6e15334f75))
* **mssql:** Fix mssql tool kind to mssql-sql ([#249](https://github.com/googleapis/genai-toolbox/issues/249)) ([1357be2](https://github.com/googleapis/genai-toolbox/commit/1357be2569b5f8d31b2b72fa83749fa8519fc8bd))
* **mysql:** Fix mysql tool kind to mysql-sql ([#248](https://github.com/googleapis/genai-toolbox/issues/248)) ([669d6b7](https://github.com/googleapis/genai-toolbox/commit/669d6b7239c36f612f02948716cf167c5a2eaa10))
* Schema float type ([#264](https://github.com/googleapis/genai-toolbox/issues/264)) ([1702f74](https://github.com/googleapis/genai-toolbox/commit/1702f74e9937eb4539c38c7152fe474870e61591))
* Typos at test cases ([#265](https://github.com/googleapis/genai-toolbox/issues/265)) ([b7c5661](https://github.com/googleapis/genai-toolbox/commit/b7c5661215c431c8590a60e029f3c340132574b7))
* Update README and quickstart with the correct async APIs. ([#269](https://github.com/googleapis/genai-toolbox/issues/269)) ([21eef2e](https://github.com/googleapis/genai-toolbox/commit/21eef2e198683d2f7fd0e606a4410b4f3a51686e))
* Update tool invoke to return json ([#266](https://github.com/googleapis/genai-toolbox/issues/266)) ([ad58cd5](https://github.com/googleapis/genai-toolbox/commit/ad58cd5855be9e1b73926e16527fb89ce778b8d9))


### Miscellaneous Chores

* **main:** Release 0.1.0 ([#272](https://github.com/googleapis/genai-toolbox/issues/272)) ([c6e3f73](https://github.com/googleapis/genai-toolbox/commit/c6e3f731e2d78b0d1043db6dbd0d69046994051d))

## [0.0.5](https://github.com/googleapis/genai-toolbox/compare/v0.0.4...v0.0.5) (2025-01-14)


### ⚠ BREAKING CHANGES

* replace Source field `ip_type` with `ipType` for consistency ([#197](https://github.com/googleapis/genai-toolbox/issues/197))
* **toolbox-sdk:** deprecate 'add_auth_headers' in favor of 'add_auth_tokens'  ([#170](https://github.com/googleapis/genai-toolbox/issues/170))

### Features

* Add support for OpenTelemetry ([#205](https://github.com/googleapis/genai-toolbox/issues/205)) ([1fcc20a](https://github.com/googleapis/genai-toolbox/commit/1fcc20a8469794ed8e6846cded44196d26c306be))
* Added Neo4j Source and Tool ([#189](https://github.com/googleapis/genai-toolbox/issues/189)) ([8a1224b](https://github.com/googleapis/genai-toolbox/commit/8a1224b9e0145c4e214d42f14f5308b508ea27ce))
* **llamaindex-sdk:** Implement OAuth support for LlamaIndex. ([#159](https://github.com/googleapis/genai-toolbox/issues/159)) ([003ce51](https://github.com/googleapis/genai-toolbox/commit/003ce510a1fb37a23e4c64fdf21376e0e32ec8ab))
* Replace Source field `ip_type` with `ipType` for consistency ([#197](https://github.com/googleapis/genai-toolbox/issues/197)) ([e069520](https://github.com/googleapis/genai-toolbox/commit/e069520bb79d086dbdd37ebc3ad9bb39b31c8fac))
* Update log with given context ([#147](https://github.com/googleapis/genai-toolbox/issues/147)) ([809e547](https://github.com/googleapis/genai-toolbox/commit/809e547a481bd4af351bbaa2dcfd203b086bb51d))


### Bug Fixes

* Correct parsing of floats/ints from json ([#180](https://github.com/googleapis/genai-toolbox/issues/180)) ([387a5b5](https://github.com/googleapis/genai-toolbox/commit/387a5b56b53ccfe0637a0f44c0ddbec8e991cc39))
* **doc:** Update example `clientId` field ([#198](https://github.com/googleapis/genai-toolbox/issues/198)) ([0c86e89](https://github.com/googleapis/genai-toolbox/commit/0c86e895066ee3dee9ab9bc20fe00934066b67ac))
* Fix config name in auth doc samples ([#186](https://github.com/googleapis/genai-toolbox/issues/186)) ([bb03457](https://github.com/googleapis/genai-toolbox/commit/bb0345767e0550fcda975958f450086e44f6a913))
* Handle shutdown gracefully ([#178](https://github.com/googleapis/genai-toolbox/issues/178)) ([66ab70f](https://github.com/googleapis/genai-toolbox/commit/66ab70f702d7178c61c8d90399483b6125ba01c8))
* Improve return error for parameters  ([#206](https://github.com/googleapis/genai-toolbox/issues/206)) ([346c57d](https://github.com/googleapis/genai-toolbox/commit/346c57da2394e398ee8cc527b84973aa2bcde642))
* **toolbox-sdk:** Deprecate 'add_auth_headers' in favor of 'add_auth_tokens'  ([#170](https://github.com/googleapis/genai-toolbox/issues/170)) ([b56fa68](https://github.com/googleapis/genai-toolbox/commit/b56fa685e379c3515025ed76d9abe61f93365a65))


### Miscellaneous Chores

* Release 0.0.5 ([#210](https://github.com/googleapis/genai-toolbox/issues/210)) ([bd407c0](https://github.com/googleapis/genai-toolbox/commit/bd407c0ab749c9a72523122a2212652f9d97ab03))

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
