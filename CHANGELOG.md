# Changelog

## [0.19.1](https://github.com/googleapis/genai-toolbox/compare/v0.18.0...v0.19.1) (2025-11-07)


### ⚠ BREAKING CHANGES

* **tools/alloydbainl:** update AlloyDB AI NL statement order ([#1753](https://github.com/googleapis/genai-toolbox/issues/1753))
* **tools/bigquery-analyze-contribution:** Add allowed dataset support ([#1675](https://github.com/googleapis/genai-toolbox/issues/1675)) ([ef28e39](https://github.com/googleapis/genai-toolbox/commit/ef28e39e90b21287ca8e69b99f4e792c78e9d31f))
* **tools/bigquery-get-dataset-info:** add allowed dataset support ([#1654](https://github.com/googleapis/genai-toolbox/issues/1654))

### Features

* Support `excludeValues` for parameters ([#1818](https://github.com/googleapis/genai-toolbox/issues/1818)) ([a8e98dc](https://github.com/googleapis/genai-toolbox/commit/a8e98dc99d208e8b37a3bc4d1ab4749b5154ed36))
* **elasticsearch:** Add Elasticsearch source and tools ([#1109](https://github.com/googleapis/genai-toolbox/issues/1109)) ([5367285](https://github.com/googleapis/genai-toolbox/commit/5367285e91ddda99ae7261d8ed4b025f975d1453))
* **mindsdb:** Add MindsDB Source and Tools  ([#878](https://github.com/googleapis/genai-toolbox/issues/878)) ([1b2cca9](https://github.com/googleapis/genai-toolbox/commit/1b2cca9faa6f7bacbeb5d25634ce3bf61067de16))
* **cloud-healthcare:** Add support for healthcare source, tool and prebuilt config ([#1853](https://github.com/googleapis/genai-toolbox/issues/1853)) ([1f833fb](https://github.com/googleapis/genai-toolbox/commit/1f833fb1a124e23819ddfec476f2e63ef31dd22f))
* **singlestore:** Add SingleStore Source and Tools ([#1333](https://github.com/googleapis/genai-toolbox/issues/1333)) ([40b9dba](https://github.com/googleapis/genai-toolbox/commit/40b9dbab088add05a66995abb1c71a9345b8f7e5))
* **source/bigquery:** Add client cache for user-passed credentials ([#1119](https://github.com/googleapis/genai-toolbox/issues/1119)) ([cf7012a](https://github.com/googleapis/genai-toolbox/commit/cf7012a82bb5c77309da3a26e563a5015786aa69))
* **source/bigquery:** Add service account impersonation support for bigquery ([#1641](https://github.com/googleapis/genai-toolbox/issues/1641)) ([e09d182](https://github.com/googleapis/genai-toolbox/commit/e09d182f88bf697a169428f477aebc9f1741e35f))
* **tools/bigquery-analyze-contribution:** Add allowed dataset support ([#1675](https://github.com/googleapis/genai-toolbox/issues/1675)) ([ef28e39](https://github.com/googleapis/genai-toolbox/commit/ef28e39e90b21287ca8e69b99f4e792c78e9d31f))
* **tools/bigquery-get-dataset-info:** Add allowed dataset support ([#1654](https://github.com/googleapis/genai-toolbox/issues/1654)) ([a2006ad](https://github.com/googleapis/genai-toolbox/commit/a2006ad57718ebad3de5c6850e9c6a5a763808ec))
* **tools/looker-run-dashboard:** New `run_dashboard` tool ([#1858](https://github.com/googleapis/genai-toolbox/issues/1858)) ([30857c2](https://github.com/googleapis/genai-toolbox/commit/30857c2294bb14961d3be49e2c368c69ecf844ec))
* **tools/looker-run-look:** Modify run_look to show query origin ([#1860](https://github.com/googleapis/genai-toolbox/issues/1860)) ([991e539](https://github.com/googleapis/genai-toolbox/commit/991e539f9c7978fa618ada3179a0b656c33ff501))
* **tools/looker:** Tools to retrieve the connections, schemas, databases, and column metadata from a looker system. ([#1804](https://github.com/googleapis/genai-toolbox/issues/1804)) ([d7d1b03](https://github.com/googleapis/genai-toolbox/commit/d7d1b03f3b746ed748d67f67e833457d55c846ab))
* **tools/mongodb:** Make MongoDB tools' `filterParams` field optional ([#1614](https://github.com/googleapis/genai-toolbox/issues/1614)) ([208ab92](https://github.com/googleapis/genai-toolbox/commit/208ab92eb377b538a99330a415ecc18790b077b7))
* **tools/neo4j-execute-cypher:** Add dry_run parameter to validate Cypher queries ([#1769](https://github.com/googleapis/genai-toolbox/issues/1769)) ([f475da6](https://github.com/googleapis/genai-toolbox/commit/f475da63ce1b65387b503ac497eca47635452723))
* **tools/postgres-list-schemas:** Add new postgres-list-schemas tool ([#1741](https://github.com/googleapis/genai-toolbox/issues/1741)) ([1a19cac](https://github.com/googleapis/genai-toolbox/commit/1a19cac7cd89ed70291eb55e190370fe7b2c1aba))
* **tools/postgres-list-views:** Add new postgres-list-views tool ([#1709](https://github.com/googleapis/genai-toolbox/issues/1709)) ([e8c7fe0](https://github.com/googleapis/genai-toolbox/commit/e8c7fe0994fedcb9be78d461fab3c98cc6bd86b2))
* **tools/serverless-spark:** Add cancel-batch tool ([#1827](https://github.com/googleapis/genai-toolbox/pull/1827))([2881683](https://github.com/googleapis/genai-toolbox/commit/28816832265250de97d84e6ba38bf6c35e040796))
* **tools/serverless-spark:** Add get_batch tool ([#1783](https://github.com/googleapis/genai-toolbox/pull/1783))([7ad1072](https://github.com/googleapis/genai-toolbox/commit/7ad10720b4638324cd77d8aa410cbd1ccf0cc93f))
* **tools/serverless-spark:** Add serverless-spark source with list_batches tool ([#1690](https://github.com/googleapis/genai-toolbox/pull/1690))([816dbce](https://github.com/googleapis/genai-toolbox/commit/816dbce268392046e54767732bd31488c6e89bdb))


### Bug Fixes

* Bigquery execute_sql to assign values to array ([#1884](https://github.com/googleapis/genai-toolbox/issues/1884)) ([559e2a2](https://github.com/googleapis/genai-toolbox/commit/559e2a22e0db20bb947702e13140ce869b5865a7))
* **cloudmonitoring:** Populate `authRequired` in tool manifest ([#1800](https://github.com/googleapis/genai-toolbox/issues/1800)) ([954152c](https://github.com/googleapis/genai-toolbox/commit/954152c7928bf0da9be221e011e32f74bc4cebbc))
* Update debug logs statements ([#1828](https://github.com/googleapis/genai-toolbox/issues/1828)) ([3cff915](https://github.com/googleapis/genai-toolbox/commit/3cff915e22c3a5e4e296607f83ae6409b896c9a9))
* Instructions to quote filters that include commas ([#1794](https://github.com/googleapis/genai-toolbox/issues/1794)) ([4b01720](https://github.com/googleapis/genai-toolbox/commit/4b0172083c0dd4c71098d4e0ab5fa0b16ea0d830))
* **source/cloud-sql-mssql:** Remove `ipAddress` field ([#1822](https://github.com/googleapis/genai-toolbox/issues/1822)) ([38d535d](https://github.com/googleapis/genai-toolbox/commit/38d535de34cfedd6828a01d9dcd25daf1bad7306))
* **tools/alloydbainl:** AlloyDB AI NL execute_sql statement order ([#1753](https://github.com/googleapis/genai-toolbox/issues/1753)) ([9723cad](https://github.com/googleapis/genai-toolbox/commit/9723cadaa181a76d8fdda65a6254f6c887c3cf57))
* **tools/postgres-execute-sql:** Do not ignore SQL failure ([#1829](https://github.com/googleapis/genai-toolbox/issues/1829)) ([8984287](https://github.com/googleapis/genai-toolbox/commit/898428759c2a1a384bea8939605cf0914d129bec))


## [0.18.0](https://github.com/googleapis/genai-toolbox/compare/v0.17.0...v0.18.0) (2025-10-23)


### Features

* Support `allowedValues`, `escape`, `minValue` and `maxValue` for parameters ([#1770](https://github.com/googleapis/genai-toolbox/issues/1770)) ([eaf7740](https://github.com/googleapis/genai-toolbox/commit/eaf77406fd386c12315d67eb685dc69e0415c516))
* **tools/looker:** Tools to allow the agent to retrieve, create, modify, and delete LookML project files. ([#1673](https://github.com/googleapis/genai-toolbox/issues/1673)) ([089081f](https://github.com/googleapis/genai-toolbox/commit/089081feb0e32f9eb65d00df5987392d413a4081))


### Bug Fixes

* **sources/mysql:** Escape mysql user agent ([#1707](https://github.com/googleapis/genai-toolbox/issues/1707)) ([eeb694c](https://github.com/googleapis/genai-toolbox/commit/eeb694c20facc40a38bfa67073c4cb1f3dd657ff))
* **sources/mysql:** Escape program_name for MySQL ([#1717](https://github.com/googleapis/genai-toolbox/issues/1717)) ([02f7f8a](https://github.com/googleapis/genai-toolbox/commit/02f7f8af979057efe99fd138cb1b958130355b68))
* **tools/http:** Allow 2xx status code on tool invocation ([#1761](https://github.com/googleapis/genai-toolbox/issues/1761)) ([a06d0d8](https://github.com/googleapis/genai-toolbox/commit/a06d0d8735fbec29bea97457248845a8c6b4aa3c))
* **tools/http:** Omit optional nil query parameters ([#1762](https://github.com/googleapis/genai-toolbox/issues/1762)) ([bd16ba3](https://github.com/googleapis/genai-toolbox/commit/bd16ba3921e6177065780e5f29870859b8e18e4f))
* **tools/looker:** Looker file content calls should not use url.QueryEscape ([#1758](https://github.com/googleapis/genai-toolbox/issues/1758)) ([336de1b](https://github.com/googleapis/genai-toolbox/commit/336de1bd04b869d322c0fd1f4667eb652159d791))


## [0.17.0](https://github.com/googleapis/genai-toolbox/compare/v0.16.0...v0.17.0) (2025-10-10)


### ⚠ BREAKING CHANGES

* **tools/bigquery-get-table-info:** add allowed dataset support ([#1093](https://github.com/googleapis/genai-toolbox/issues/1093))
* **tools/bigquery-list-dataset-ids:** add allowed datasets support ([#1573](https://github.com/googleapis/genai-toolbox/issues/1573))

### Features

* Add configs and workflows for docs versioning ([#1611](https://github.com/googleapis/genai-toolbox/issues/1611)) ([21ac98b](https://github.com/googleapis/genai-toolbox/commit/21ac98bc065e95bde911d66185c67d8380891bf8))
* Add metadata in MCP Manifest for Toolbox auth ([#1395](https://github.com/googleapis/genai-toolbox/issues/1395)) ([0b3dac4](https://github.com/googleapis/genai-toolbox/commit/0b3dac41322f7aaa5a19df571686fa8fd4338ca5))
* Add program name to MySQL connections ([#1617](https://github.com/googleapis/genai-toolbox/issues/1617)) ([c4a22b8](https://github.com/googleapis/genai-toolbox/commit/c4a22b8d3bd0307325215ebd2f30ba37927cd37e))
* **source/bigquery:** Add optional write mode config ([#1157](https://github.com/googleapis/genai-toolbox/issues/1157)) ([63adc78](https://github.com/googleapis/genai-toolbox/commit/63adc78beae949dfe5e300c50e5ceef073e9652c))
* **sources/alloydb,cloudsqlpg,cloudsqlmysql,cloudsqlmssql:** Support PSC connection ([#1686](https://github.com/googleapis/genai-toolbox/issues/1686)) ([9d2bf79](https://github.com/googleapis/genai-toolbox/commit/9d2bf79becfda104ef77f34b8d4b22cbedbc4bf3))
* **sources/mssql:** Add app name to MSSQL ([#1620](https://github.com/googleapis/genai-toolbox/issues/1620)) ([1536d1f](https://github.com/googleapis/genai-toolbox/commit/1536d1fdabb9d7f73dbdeebeb05a83d9a3b78e1c))
* **sources/oracle:** Add Oracle Source and Tool ([#1456](https://github.com/googleapis/genai-toolbox/issues/1456)) ([3a19a50](https://github.com/googleapis/genai-toolbox/commit/3a19a50ff211e33429de1d05338d353359a52987))
* **sources/oracle:** Switch Oracle driver from godror to go-ora ([#1685](https://github.com/googleapis/genai-toolbox/issues/1685)) ([8faf376](https://github.com/googleapis/genai-toolbox/commit/8faf37667e371b4ed88ebb892e8784b67611ba64))
* **tool/bigquery-list-dataset-ids:** Add allowed datasets support ([#1573](https://github.com/googleapis/genai-toolbox/issues/1573)) ([1a44c67](https://github.com/googleapis/genai-toolbox/commit/1a44c671ec593e764a2d2f67f70a98ceec20a168))
* **tools/bigquery-get-table-info:** Add allowed dataset support ([#1093](https://github.com/googleapis/genai-toolbox/issues/1093)) ([acb205c](https://github.com/googleapis/genai-toolbox/commit/acb205ca4761d59ce97b804827230978c8c98ede))
* **tools/dataform:** Add dataform compile tool ([#1470](https://github.com/googleapis/genai-toolbox/issues/1470)) ([3be9b7b](https://github.com/googleapis/genai-toolbox/commit/3be9b7b3bdf112fe7303706e56e9f39935cde661))
* **tools/looker:** Add support for pulse, vacuum and analyze audit and performance functions on a Looker instance ([#1581](https://github.com/googleapis/genai-toolbox/issues/1581)) ([5aed4e1](https://github.com/googleapis/genai-toolbox/commit/5aed4e136d0091731d2ded10ec076ee789e1987c))
* **tools/looker:** Enable access to the Conversational Analytics API for Looker ([#1596](https://github.com/googleapis/genai-toolbox/issues/1596)) ([2d5a93e](https://github.com/googleapis/genai-toolbox/commit/2d5a93e312990c8a9f3170c7e9c655f97cf11712))


### Bug Fixes

* Added google_ml_integration extension to use alloydb ai-nl support api ([#1445](https://github.com/googleapis/genai-toolbox/issues/1445)) ([dbc477a](https://github.com/googleapis/genai-toolbox/commit/dbc477ab0f832495cf51f73ea16ae363472d6eed))
* Fix broken links ([#1625](https://github.com/googleapis/genai-toolbox/issues/1625)) ([36c6584](https://github.com/googleapis/genai-toolbox/commit/36c658472ccdeb6cddd8a4452a8b3438aeb0a744))
* Remove duplicated build type in Dockerfile ([#1598](https://github.com/googleapis/genai-toolbox/issues/1598)) ([b43c945](https://github.com/googleapis/genai-toolbox/commit/b43c94575d86aa65b0528d59f9b41d30b439fee5))
* **source/bigquery:** Allowed datasets project id issue with client oauth ([#1663](https://github.com/googleapis/genai-toolbox/issues/1663)) ([f4cf486](https://github.com/googleapis/genai-toolbox/commit/f4cf486fa929299fef076cf71689776e5dec19c1))
* **sources/looker:** Allow Looker to be configured without setting a Client Id or Secret ([#1496](https://github.com/googleapis/genai-toolbox/issues/1496)) ([67d8221](https://github.com/googleapis/genai-toolbox/commit/67d8221a2e780df54a81f0f7e8f7e41e4bf1a82e))
* **tools/looker:** Refactor run-inline-query logic to helper function ([#1497](https://github.com/googleapis/genai-toolbox/issues/1497)) ([62af39d](https://github.com/googleapis/genai-toolbox/commit/62af39d751443eb758586663969b162c868a233f))
* **tools/mysql-list-tables:** Update sql query to resolve subquery scope error ([#1629](https://github.com/googleapis/genai-toolbox/issues/1629)) ([94e19d8](https://github.com/googleapis/genai-toolbox/commit/94e19d87e54e831b80eb766572e48bc7370305d8))

## [0.16.0](https://github.com/googleapis/genai-toolbox/compare/v0.15.0...v0.16.0) (2025-09-25)


### ⚠ BREAKING CHANGES

* **tool/bigquery-execute-sql:** add allowed datasets support ([#1443](https://github.com/googleapis/genai-toolbox/issues/1443))
* **tool/bigquery-forecast:** add allowed datasets support ([#1412](https://github.com/googleapis/genai-toolbox/issues/1412))

### Features

* **cassandra:** Add Cassandra Source and Tool ([#1012](https://github.com/googleapis/genai-toolbox/issues/1012)) ([6e42053](https://github.com/googleapis/genai-toolbox/commit/6e420534ee894da4a8d226acb6cdb63d0d5d9ce5))
* **sources/postgres:** Add application_name ([#1504](https://github.com/googleapis/genai-toolbox/issues/1504)) ([72a2366](https://github.com/googleapis/genai-toolbox/commit/72a2366b28870aa6f81c4f890f4770ec5ecffdba))
* **tool/bigquery-execute-sql:** Add allowed datasets support ([#1443](https://github.com/googleapis/genai-toolbox/issues/1443)) ([9501ebb](https://github.com/googleapis/genai-toolbox/commit/9501ebbdbcba871b98663185c690308dda1729b5))
* **tool/bigquery-forecast:** Add allowed datasets support ([#1412](https://github.com/googleapis/genai-toolbox/issues/1412)) ([88bac7e](https://github.com/googleapis/genai-toolbox/commit/88bac7e36f5ebb6ad18773bff30b85ef678431e7))
* **tools/clickhouse-list-tables:** Add list-tables tool ([#1446](https://github.com/googleapis/genai-toolbox/issues/1446)) ([69a3caf](https://github.com/googleapis/genai-toolbox/commit/69a3cafabec5a40e2776d71de3587c0d16c722a2))


### Bug Fixes

* **tool/mongodb-find:** Fix find tool `limit` field ([#1570](https://github.com/googleapis/genai-toolbox/issues/1570)) ([4166bf7](https://github.com/googleapis/genai-toolbox/commit/4166bf7ab85732f64b877d5f20235057df919049))
* **tools/mongodb:** Concat filter params only once in mongodb update tools ([#1545](https://github.com/googleapis/genai-toolbox/issues/1545)) ([295f9dc](https://github.com/googleapis/genai-toolbox/commit/295f9dc41a43f0a4bdbd99e465bf2be01249084e))

## [0.15.0](https://github.com/googleapis/genai-toolbox/compare/v0.14.0...v0.15.0) (2025-09-18)


### ⚠ BREAKING CHANGES

* **prebuilt:** update prebuilt tool names to use consistent guidance ([#1421](https://github.com/googleapis/genai-toolbox/issues/1421))
* **tools/alloydb-wait-for-operation:** Add `alloydb-admin` source to `alloydb-wait-for-operation` tool ([#1449](https://github.com/googleapis/genai-toolbox/issues/1449))

### Features

* Add AlloyDB admin source ([#1369](https://github.com/googleapis/genai-toolbox/issues/1369)) ([33beb71](https://github.com/googleapis/genai-toolbox/commit/33beb7187d2e0f968fc949a00c780073d1bc7cdd))
* Add Cloud monitoring source and tool ([#1311](https://github.com/googleapis/genai-toolbox/issues/1311)) ([d661f53](https://github.com/googleapis/genai-toolbox/commit/d661f5343f2ad28fbf0481db16440aec823eece6))
* Add YugabyteDB Source and Tool ([#732](https://github.com/googleapis/genai-toolbox/issues/732)) ([664711f](https://github.com/googleapis/genai-toolbox/commit/664711f4b35409bd1c57af92f625b70a0dc9a4e6))
* **prebuilt:** Update default values for prebuilt tools ([#1355](https://github.com/googleapis/genai-toolbox/issues/1355)) ([70e832b](https://github.com/googleapis/genai-toolbox/commit/70e832bd08a98a95b925e590f31c8d3f2d8b6aa0))
* **prebuilt/cloud-sql:** Add  list instances tool for cloudsql ([#1310](https://github.com/googleapis/genai-toolbox/issues/1310)) ([0171228](https://github.com/googleapis/genai-toolbox/commit/01712284b480774ffa68930affae290ee2e3fcfd))
* **prebuilt/cloud-sql:** Add cloud sql create database tool. ([#1453](https://github.com/googleapis/genai-toolbox/issues/1453)) ([a1bc044](https://github.com/googleapis/genai-toolbox/commit/a1bc04477b0f822ffaab039098682f1776b8a472))
* **prebuilt/cloud-sql:** Add `cloud-sql-get-instances` tool ([#1383](https://github.com/googleapis/genai-toolbox/issues/1383)) ([77919c7](https://github.com/googleapis/genai-toolbox/commit/77919c7d8e4aac16eeb703c0cc61ca774dc4f94e))
* **prebuilt/cloud-sql:** Add create user tool for cloud sql ([#1406](https://github.com/googleapis/genai-toolbox/issues/1406)) ([3a6b517](https://github.com/googleapis/genai-toolbox/commit/3a6b51752f077b225b8c2e2e7308a69a68eec3c0))
* **prebuilt/cloud-sql:** Add list databases tool for cloud sql ([#1454](https://github.com/googleapis/genai-toolbox/issues/1454)) ([e6a6c61](https://github.com/googleapis/genai-toolbox/commit/e6a6c615d5480e8930ad173d44d243f5bd99eebc))
* **prebuilt/cloud-sql:** Package cloud sql tools ([#1455](https://github.com/googleapis/genai-toolbox/issues/1455)) ([bf6266b](https://github.com/googleapis/genai-toolbox/commit/bf6266ba1131bd1c5829ac112a8c45c8a5919fea))
* **prebuilt/cloud-sql-mssql:** Add create instance tool for mssql ([#1440](https://github.com/googleapis/genai-toolbox/issues/1440)) ([b176523](https://github.com/googleapis/genai-toolbox/commit/b17652309d8a02b1f20c6c576b1617b23c8e481f))
* **prebuilt/cloud-sql-mysql:** Add create instance tool for Cloud SQL MySQL ([#1434](https://github.com/googleapis/genai-toolbox/issues/1434)) ([15b628d](https://github.com/googleapis/genai-toolbox/commit/15b628d2d2feb2ecdd418394b9265a6c77c77f6d))
* **prebuilt/cloud-sql-mysql:** Add env var support for IP Type ([#1232](https://github.com/googleapis/genai-toolbox/issues/1232)) ([#1347](https://github.com/googleapis/genai-toolbox/issues/1347)) ([0cd3f16](https://github.com/googleapis/genai-toolbox/commit/0cd3f16f877f426b45e35625ba0af03789459591))
* **prebuilt/cloudsqlpg:** Add cloud sql pg create instance tool ([#1403](https://github.com/googleapis/genai-toolbox/issues/1403)) ([d302499](https://github.com/googleapis/genai-toolbox/commit/d30249961b5a2ddc2c3809b481085d1ca034ead0))
* **prebuilt/mysql:** Add a new tool to show query plan of a given query in MySQL ([#1474](https://github.com/googleapis/genai-toolbox/issues/1474)) ([1a42e05](https://github.com/googleapis/genai-toolbox/commit/1a42e05675645ac4f1b89edef7a71ac61b637a76))
* **prebuilt/mysql:** Add  `queryParams` field in MySQL prebuilt config ([#1318](https://github.com/googleapis/genai-toolbox/issues/1318)) ([4b32c2a](https://github.com/googleapis/genai-toolbox/commit/4b32c2a7701ce5ccc56d019055283e73e7046372))
* **prebuilt/neo4j:** Add prebuiltconfig support for neo4j ([#1352](https://github.com/googleapis/genai-toolbox/issues/1352)) ([f819e26](https://github.com/googleapis/genai-toolbox/commit/f819e2644315a589ec283494f244c1b8407cae59))
* **prebuilt/observability:** Add cloud sql observability tools ([#1425](https://github.com/googleapis/genai-toolbox/issues/1425)) ([236be89](https://github.com/googleapis/genai-toolbox/commit/236be89961fe423c1ec992d3d1f699f77a6e5b29))
* **prebuilt/postgres:** Add postgres prebuilt tools ([#1473](https://github.com/googleapis/genai-toolbox/issues/1473)) ([edca9dc](https://github.com/googleapis/genai-toolbox/commit/edca9dc7d772baf1a234485020fa69d76f71bfcc))
* **prebuilt/sqlite:** Prebuilt tools for the sqlite. ([#1227](https://github.com/googleapis/genai-toolbox/issues/1227)) ([681c2b4](https://github.com/googleapis/genai-toolbox/commit/681c2b4f3a65837d972c138c623c08fb6b1f1785))
* **source/alloydb-admin:** Add user agent and attach alloydb api in `alloydb-admin` source ([#1448](https://github.com/googleapis/genai-toolbox/issues/1448)) ([9710014](https://github.com/googleapis/genai-toolbox/commit/971001400f25796784f8aeb3ec5cb1a2df2e4c69))
* **source/bigquery:** Add support for datasets selection ([#1313](https://github.com/googleapis/genai-toolbox/issues/1313)) ([aa39724](https://github.com/googleapis/genai-toolbox/commit/aa3972470fd0f6f5901c5d85dd05f1e2ae973e7b))
* **source/cloud-monitoring:** Add support for user agent in cloud monitoring source ([#1472](https://github.com/googleapis/genai-toolbox/issues/1472)) ([92680b1](https://github.com/googleapis/genai-toolbox/commit/92680b18d6159300ae66f80ddb4c6bf0547d45a1))
* **source/cloud-sql-admin:** Add User agent and attach sqldmin in `cloud-sql-admin` source. ([#1441](https://github.com/googleapis/genai-toolbox/issues/1441)) ([56b6574](https://github.com/googleapis/genai-toolbox/commit/56b6574fc2c506c7c7df7f2a25686e3e4aae0e8a))
* **source/cloudsqladmin:** Add cloud sql admin source ([#1408](https://github.com/googleapis/genai-toolbox/issues/1408)) ([4f46782](https://github.com/googleapis/genai-toolbox/commit/4f4678292762507494515ce61188cd0310805c40))
* **tool/cloudsql:** Add cloud sql wait for operation tool with exponential backoff ([#1306](https://github.com/googleapis/genai-toolbox/issues/1306)) ([3aef2bb](https://github.com/googleapis/genai-toolbox/commit/3aef2bb7be8274bb4718739faeaa5f97b50dbf19))
* **tools/alloydb-create-cluster:** Add custom tool kind for AlloyDB create cluster ([#1331](https://github.com/googleapis/genai-toolbox/issues/1331)) ([76bb876](https://github.com/googleapis/genai-toolbox/commit/76bb876d546780908c1a69ef3b1a92781af28a3b))
* **tools/alloydb-create-instance:** Add new custom tool kind for AlloyDB ([#1379](https://github.com/googleapis/genai-toolbox/issues/1379)) ([091cd9a](https://github.com/googleapis/genai-toolbox/commit/091cd9aa1aabe1cb3de2ce2be5c707a8f77ad647))
* **tools/alloydb-create-user:** Add new custom tool kind for AlloyDB create user ([#1380](https://github.com/googleapis/genai-toolbox/issues/1380)) ([ab3fd26](https://github.com/googleapis/genai-toolbox/commit/ab3fd261af373dcdaf4555292c63d7095d7a02df))
* **tools/alloydb-get-cluster:** Add new tool for AlloyDB ([#1420](https://github.com/googleapis/genai-toolbox/issues/1420)) ([c181dab](https://github.com/googleapis/genai-toolbox/commit/c181dabc91bdc1c24c89a3c7bba0049d9af4cf2b))
* **tools/alloydb-get-instance:** Add new for AlloyDB ([#1435](https://github.com/googleapis/genai-toolbox/issues/1435)) ([f2d9e3b](https://github.com/googleapis/genai-toolbox/commit/f2d9e3b57963082f0db70880d5c02b1cbe3eb75d))
* **tools/alloydb-get-user:** Add new tool for AlloyDB ([#1436](https://github.com/googleapis/genai-toolbox/issues/1436)) ([677254e](https://github.com/googleapis/genai-toolbox/commit/677254e6d9c532fa9f0fb0b0e4062446640ab75f))
* **tools/alloydb-list-cluster:** Add custom tool kind for AlloyDB ([#1319](https://github.com/googleapis/genai-toolbox/issues/1319)) ([d4a9eb0](https://github.com/googleapis/genai-toolbox/commit/d4a9eb0ce217c7969aff61868e53c7dc7757d28d))
* **tools/alloydb-list-instances:** Add custom tool kind for AlloyDB ([#1357](https://github.com/googleapis/genai-toolbox/issues/1357)) ([93c1b30](https://github.com/googleapis/genai-toolbox/commit/93c1b30fced113d6721ade9fdcfb92b5ed6c0ad6))
* **tools/alloydb-list-users:** Add new custom tool kind for AlloyDB ([#1377](https://github.com/googleapis/genai-toolbox/issues/1377)) ([3a8a65c](https://github.com/googleapis/genai-toolbox/commit/3a8a65ceaa92368563e237087c4a38ed7c0d3fd5))
* **tools/bigquery-analyze-contribution:** Add analyze contribution tool ([#1223](https://github.com/googleapis/genai-toolbox/issues/1223)) ([81d239b](https://github.com/googleapis/genai-toolbox/commit/81d239b053a6978250878a6809905dcc9424909e))
* **tools/bigquery-conversational-analytics:** Add allowed datasets support ([#1411](https://github.com/googleapis/genai-toolbox/issues/1411)) ([345bd6a](https://github.com/googleapis/genai-toolbox/commit/345bd6af520bb9ce8a43834951e68fea7bbe6a02))
* **tools/bigquery-search-catalog:** Add new tool to BigQuery ([#1382](https://github.com/googleapis/genai-toolbox/issues/1382)) ([bffb39d](https://github.com/googleapis/genai-toolbox/commit/bffb39dea3cc946a1e611e3523241443b1e4f047))
* **tools/bigquery:** Add `useClientOAuth` to BigQuery prebuilt source config ([#1431](https://github.com/googleapis/genai-toolbox/issues/1431)) ([fe2999a](https://github.com/googleapis/genai-toolbox/commit/fe2999a691ac92b2bf35cb7cfd504df2f3ce84b3))
* **tools/clickhouse-list-databases:** Add `list-databases` tool to clickhouse source ([#1274](https://github.com/googleapis/genai-toolbox/issues/1274)) ([e515d92](https://github.com/googleapis/genai-toolbox/commit/e515d9254f3b8e89f89322d490eb3cedce85d2bb))
* **tools/firestore-get-rules:** Add `databaseId` to the Firestore source and `firestore-get-rules` tool ([#1505](https://github.com/googleapis/genai-toolbox/issues/1505)) ([7450482](https://github.com/googleapis/genai-toolbox/commit/7450482bb2479eab7d1c8f0d40755a8d11aa3b26))
* **tools/firestore:** Add `firestore-query` tool ([#1305](https://github.com/googleapis/genai-toolbox/issues/1305)) ([cce602f](https://github.com/googleapis/genai-toolbox/commit/cce602f28097353f6a3017cec1fa5f75283f111d))
* **tools/looker:** Query tracking for MCP Toolbox in Looker System Activity views ([#1410](https://github.com/googleapis/genai-toolbox/issues/1410)) ([2036c8e](https://github.com/googleapis/genai-toolbox/commit/2036c8efd2fb9edc26df599629d3131c6c367f4b))
* **tools/mssql-list-tables:** Add new tool for sql server ([#1433](https://github.com/googleapis/genai-toolbox/issues/1433)) ([b036047](https://github.com/googleapis/genai-toolbox/commit/b036047a21f63265c9d9637ac1a671792c9c2e80))
* **tools/mysql-list-active-queries:** Add a new tool to list ongoing queries in a MySQL instance ([#1471](https://github.com/googleapis/genai-toolbox/issues/1471)) ([ed54cd6](https://github.com/googleapis/genai-toolbox/commit/ed54cd6cfd17a3bdd84025d4eb8264763da36a98))
* **tools/mysql-list-table-fragmentation:** Add a new tool to list table fragmentation in a MySQL instance ([#1479](https://github.com/googleapis/genai-toolbox/issues/1479)) ([fe651d8](https://github.com/googleapis/genai-toolbox/commit/fe651d822f88832833e869ec049c6c084eae7e51))
* **tools/mysql-list-tables-missing-index:** Add a new tool to list tables that do not have primary or unique keys in a MySQL instance ([#1493](https://github.com/googleapis/genai-toolbox/issues/1493)) ([9eb821a](https://github.com/googleapis/genai-toolbox/commit/9eb821a6dca408ba993f904aa42b5b4f70674ba7))
* **tools/mysql-list-tables:** Add new tool for MySQL ([#1287](https://github.com/googleapis/genai-toolbox/issues/1287)) ([6c8460b](https://github.com/googleapis/genai-toolbox/commit/6c8460b0e507315d407c91ba1c821f4820cc1620))
* **tools/postgres-list-active-queries:** Add new `postgres-list-active-queries` tool ([#1400](https://github.com/googleapis/genai-toolbox/issues/1400)) ([b2b06c7](https://github.com/googleapis/genai-toolbox/commit/b2b06c72c29fd99a0c7118b85e6f7bcf6853d173))
* **tools/postgres-list-tables:** Add new tool to postgres source ([#1284](https://github.com/googleapis/genai-toolbox/issues/1284)) ([71f360d](https://github.com/googleapis/genai-toolbox/commit/71f360d31522f429a646b705ce7d1d11dac4cf68))
* **tools/spanner-list-tables:** Add new tool `spanner-list-tables` ([#1404](https://github.com/googleapis/genai-toolbox/issues/1404)) ([7d384dc](https://github.com/googleapis/genai-toolbox/commit/7d384dc28f8c37dddc2f6cefc0bbeb4c201e3167))


### Bug Fixes

* **bigquery:** Add `Bearer` parsing to auth token ([#1386](https://github.com/googleapis/genai-toolbox/issues/1386)) ([b5f9780](https://github.com/googleapis/genai-toolbox/commit/b5f9780a59e15eca2591dee32f5da42435e03039))
* **source/alloydb-admin, source/cloudsql-admin:** Post append new user agent ([#1494](https://github.com/googleapis/genai-toolbox/issues/1494)) ([30f1d3a](https://github.com/googleapis/genai-toolbox/commit/30f1d3a983aa317f1e1a98f9fe753005b56c52bd))
* **tools/alloydb:** Update parameter names and set default description for AlloyDB control plane tools ([#1468](https://github.com/googleapis/genai-toolbox/issues/1468)) ([6c140d7](https://github.com/googleapis/genai-toolbox/commit/6c140d718a66b45c7ec2d5a267331adb7680f689))
* **tools/bigquery-conversational-analytics:** Fix authentication scope error in Cloud Run ([#1381](https://github.com/googleapis/genai-toolbox/issues/1381)) ([80b7488](https://github.com/googleapis/genai-toolbox/commit/80b7488ad248ab1d98ee6713e1f6737f67f6754b))
* **tools/mysql-list-tables:** Update `mysql-list-tables` table_names parameter with default value ([#1439](https://github.com/googleapis/genai-toolbox/issues/1439)) ([da24661](https://github.com/googleapis/genai-toolbox/commit/da246610e105df10a9dc1bce19fa35d408c039f3))
* **tools/neo4j:** Implement value conversion for Neo4j types to JSON-compatible ([#1428](https://github.com/googleapis/genai-toolbox/issues/1428)) ([4babc4e](https://github.com/googleapis/genai-toolbox/commit/4babc4e11b3b64db8d8c9d6b65e47744f5174f7f))

## [0.14.0](https://github.com/googleapis/genai-toolbox/compare/v0.13.0...v0.14.0) (2025-09-05)


### ⚠ BREAKING CHANGES

* **bigquery:** Move `useClientOAuth` config from tool to source ([#1279](https://github.com/googleapis/genai-toolbox/issues/1279)) ([8d20a48](https://github.com/googleapis/genai-toolbox/commit/8d20a48f13bcda853d41bdf80a162de12b076d1b))
* **tools/bigquerysql:** remove `useClientOAuth` from tools config ([#1312](https://github.com/googleapis/genai-toolbox/issues/1312))

### Features

* **clickhouse:** Add ClickHouse Source and Tools ([#1088](https://github.com/googleapis/genai-toolbox/issues/1088)) ([75a04a5](https://github.com/googleapis/genai-toolbox/commit/75a04a55dd2259bed72fe95119a7a51a906c0b21))
* **prebuilt/alloydb-postgres:** Support ipType and IAM users ([#1324](https://github.com/googleapis/genai-toolbox/issues/1324)) ([0b2121e](https://github.com/googleapis/genai-toolbox/commit/0b2121ea72eb81348dcd9c740a62ccd32e71fe37))
* **server/mcp:** Support toolbox auth in mcp ([#1140](https://github.com/googleapis/genai-toolbox/issues/1140)) ([ca353e0](https://github.com/googleapis/genai-toolbox/commit/ca353e0b66fedc00e9110df57db18632aef49018))
* **source/mysql:** Support `queryParams` in MySQL source ([#1299](https://github.com/googleapis/genai-toolbox/issues/1299)) ([3ae2526](https://github.com/googleapis/genai-toolbox/commit/3ae2526e0fe36b57b05a9b54f1d99f3fc68d9657))
* **tools/bigquery:** Support end-user credential passthrough on multiple BQ tools ([#1314](https://github.com/googleapis/genai-toolbox/issues/1314)) ([88f4b30](https://github.com/googleapis/genai-toolbox/commit/88f4b3028df3b6a400936cdf8a035bf55021924c))
* **tools/looker:** Add description for looker-get-models tool ([#1266](https://github.com/googleapis/genai-toolbox/issues/1266)) ([89af3a4](https://github.com/googleapis/genai-toolbox/commit/89af3a4ca332f029615b2a739d1f6cd50519638d))
* **tools/looker:** Authenticate via end user credentials ([#1257](https://github.com/googleapis/genai-toolbox/issues/1257)) ([8755e3d](https://github.com/googleapis/genai-toolbox/commit/8755e3db3476abb35629b3cca9c78db7366757a4))
* **tools/looker:** Report field suggestions to agent ([#1267](https://github.com/googleapis/genai-toolbox/issues/1267)) ([2cad82e](https://github.com/googleapis/genai-toolbox/commit/2cad82e5107566dd6c9b75e34e9976af63af0bb5))


### Bug Fixes

* Do not print usage on runtime error ([#1315](https://github.com/googleapis/genai-toolbox/issues/1315)) ([afba7a5](https://github.com/googleapis/genai-toolbox/commit/afba7a57cdd4fe7c1b0741dbf8f8c78b14a68089))
* Update env var to allow empty string ([#1260](https://github.com/googleapis/genai-toolbox/issues/1260)) ([03aa9fa](https://github.com/googleapis/genai-toolbox/commit/03aa9fabacda06f860c9f178485126bddb7d5782))
* **tools/firestore:** Add document/collection path validation ([#1229](https://github.com/googleapis/genai-toolbox/issues/1229)) ([14c2249](https://github.com/googleapis/genai-toolbox/commit/14c224939a2f9bb349fa00a7d5227877198530c2))
* **tools/looker-get-dashboards:** Fix Looker client OAuth check ([#1338](https://github.com/googleapis/genai-toolbox/issues/1338)) ([36225aa](https://github.com/googleapis/genai-toolbox/commit/36225aa6db7f8426ad87930866530fde4e9bf0cd))
* **tools/oceanbase:** Fix encoded text with mysql driver ([#1283](https://github.com/googleapis/genai-toolbox/issues/1283)) ([d16f89f](https://github.com/googleapis/genai-toolbox/commit/d16f89fbb6e49c03998f114ef7dc2b584b5e4967)), closes [#1161](https://github.com/googleapis/genai-toolbox/issues/1161)

## [0.13.0](https://github.com/googleapis/genai-toolbox/compare/v0.12.0...v0.13.0) (2025-08-27)

### ⚠ BREAKING CHANGES

* **prebuilt/alloydb:** Add bearer token support for alloydb-wait-for-operation ([#1183](https://github.com/googleapis/genai-toolbox/issues/1183))


### Features

* Add capability to set default for environment variable in config ([#1248](https://github.com/googleapis/genai-toolbox/issues/1248)) ([5bcd52e](https://github.com/googleapis/genai-toolbox/commit/5bcd52e7dcd0773ded723585f4abe29d044e1540))
* **firebird:** Add Firebird SQL 2.5+ source and tool ([#1011](https://github.com/googleapis/genai-toolbox/issues/1011)) ([4f6b806](https://github.com/googleapis/genai-toolbox/commit/4f6b806de947efc4e12bdb50dff7781aedb7b966))
* **oceanbase:** Add Oceanbase source and tool  ([#895](https://github.com/googleapis/genai-toolbox/issues/895)) ([6fc4982](https://github.com/googleapis/genai-toolbox/commit/6fc49826d43f46c84028e752ebebddf3d94b3d13))
* **server/mcp:** Support `ping` mechanism ([#1178](https://github.com/googleapis/genai-toolbox/issues/1178)) ([5dcc66c](https://github.com/googleapis/genai-toolbox/commit/5dcc66c84fa72c75ec50a9ac5198018212ec2979))
* **server:** Fail-fast on environment variable substitution ([#1177](https://github.com/googleapis/genai-toolbox/issues/1177)) ([212aaba](https://github.com/googleapis/genai-toolbox/commit/212aaba74c8b431de8a5f7b9822a0af4afcaaa0e))
* **server:** Implement Tool call auth error propagation ([#1235](https://github.com/googleapis/genai-toolbox/issues/1235)) ([b94a021](https://github.com/googleapis/genai-toolbox/commit/b94a021ca11c6637cf8038449483b5e75f2012b3))
* **sources/bigquery:** Add support for user-credential passthrough  ([#1067](https://github.com/googleapis/genai-toolbox/issues/1067)) ([650e2e2](https://github.com/googleapis/genai-toolbox/commit/650e2e26f51bff75ce66343f64944d0a89a58b69))
* **tool/looker:** Add support for `description` field in looker tool ([#1199](https://github.com/googleapis/genai-toolbox/issues/1199)) ([97f0dd2](https://github.com/googleapis/genai-toolbox/commit/97f0dd2acf26caf28ecad65abea8779c196a27f1))
* **tools/bigquery-ask-data-insights:** Add bigquery `ask-data-insights` tool ([#932](https://github.com/googleapis/genai-toolbox/issues/932)) ([7651357](https://github.com/googleapis/genai-toolbox/commit/7651357d424a2b6656d8b6818cebc5c8a86ed053))
* **tools/bigquery-forecast:** Add bigqueryforecast tool ([#1148](https://github.com/googleapis/genai-toolbox/issues/1148)) ([2ad0ccf](https://github.com/googleapis/genai-toolbox/commit/2ad0ccf83df542340087742468d6762f81eedee6))
* **tools/firestore-add-documents:** Add firestore-add-documents tool ([#1107](https://github.com/googleapis/genai-toolbox/issues/1107)) ([ee4a70a](https://github.com/googleapis/genai-toolbox/commit/ee4a70a0e82b346b07b5b4c60dfa060da2273f50))
* **tools/firestore-update-document:** Add firestore-update-document tool ([#1191](https://github.com/googleapis/genai-toolbox/issues/1191)) ([0010123](https://github.com/googleapis/genai-toolbox/commit/00101232a39c70288aac5715649c184858d351e3))
* **tools/looker:** Control over whether hidden objects are surfaced ([#1222](https://github.com/googleapis/genai-toolbox/issues/1222)) ([bc91559](https://github.com/googleapis/genai-toolbox/commit/bc91559cc4e5b20385b84cc562b624fabf7e47a8))
* **trino:** Add Trino source and tools ([#948](https://github.com/googleapis/genai-toolbox/issues/948)) ([7dd123b](https://github.com/googleapis/genai-toolbox/commit/7dd123b3d76b8eb2b74b5d960959c1f90684b37e))


### Bug Fixes

* **tools/looker:** Lookergetdashboards uses proper Authorized helper func ([#1255](https://github.com/googleapis/genai-toolbox/issues/1255)) ([00866bc](https://github.com/googleapis/genai-toolbox/commit/00866bc7fc33115c547213e60316ae889735fdbb))
* **tools/mongodb-find-one:** ProjectPayload unmarshaling ([#1167](https://github.com/googleapis/genai-toolbox/issues/1167)) ([8ea6a98](https://github.com/googleapis/genai-toolbox/commit/8ea6a98bd9096ba97722e5f807366887e864004f))
* **tools/mysql:** Fix encoded text for mysql ([#1161](https://github.com/googleapis/genai-toolbox/issues/1161)) ([a37cfa8](https://github.com/googleapis/genai-toolbox/commit/a37cfa841d151b9995d4fab73cfc5e4d30d2cc57)), closes [#840](https://github.com/googleapis/genai-toolbox/issues/840)

## [0.12.0](https://github.com/googleapis/genai-toolbox/compare/v0.11.0...v0.12.0) (2025-08-14)


### Features

* **prebuiltconfig:** Introduce additional parameter to limit context in list_tables ([#1151](https://github.com/googleapis/genai-toolbox/issues/1151)) ([497d3b1](https://github.com/googleapis/genai-toolbox/commit/497d3b126da252a4b59806ca2ca3c56e78efc13d))
* **prebuiltconfig/alloydb-admin:** Add list cluster, instance and users ([#1126](https://github.com/googleapis/genai-toolbox/issues/1126)) ([b42c139](https://github.com/googleapis/genai-toolbox/commit/b42c139158650fb1f3b696965e840c52e2016bf0))
* **prebuiltconfig/alloydb-admin:** Add tool to create user via Built in user type or IAM ([#1130](https://github.com/googleapis/genai-toolbox/issues/1130)) ([f5bcb9c](https://github.com/googleapis/genai-toolbox/commit/f5bcb9c755a2c1747d0beeda568b6217d7420e7a))
* **source/http:** Add User Agent to `http` invocations ([#1102](https://github.com/googleapis/genai-toolbox/issues/1102)) ([6f55b78](https://github.com/googleapis/genai-toolbox/commit/6f55b78e96b8c7aa9aca601cfae4d62f3e1eb42b))
* **sources/postgres:** Add support for `queryParams` ([#1047](https://github.com/googleapis/genai-toolbox/issues/1047)) ([7b57251](https://github.com/googleapis/genai-toolbox/commit/7b5725140279de21fece45e860945b7a7d23e7d0)), closes [#963](https://github.com/googleapis/genai-toolbox/issues/963)
* **tools/bigquery-execute-sql:** Add dry run support ([#1057](https://github.com/googleapis/genai-toolbox/issues/1057)) ([1cac9b5](https://github.com/googleapis/genai-toolbox/commit/1cac9b5b378153c7dc65ff3dfb4ebd852b715a10))
* **tools/dataplex-search-aspect-types:** Add support for `dataplex-search-aspect-types` tool ([#1061](https://github.com/googleapis/genai-toolbox/issues/1061)) ([d940187](https://github.com/googleapis/genai-toolbox/commit/d940187c851666cc201f519665fb4f2e1478465c))
* **tools/looker:** Add `looker-make-look` tool to create Looks ([#1099](https://github.com/googleapis/genai-toolbox/issues/1099)) ([61d9489](https://github.com/googleapis/genai-toolbox/commit/61d94893448f633a5f2b9d7f0744ab40704af824))
* **tools/looker:** Add visualizations to `query-url` tool ([#1090](https://github.com/googleapis/genai-toolbox/issues/1090)) ([5bf2758](https://github.com/googleapis/genai-toolbox/commit/5bf275846a268a8d305d6392fa4e8e79e365f00d))
* **tools/looker:** New Looker tools for dashboards ([#1118](https://github.com/googleapis/genai-toolbox/issues/1118)) ([42be3f5](https://github.com/googleapis/genai-toolbox/commit/42be3f550ceab34baf43fe2a246ded7a09cff8e3))
* **ui:** Add login with google button for automatic id token retrieval ([#1044](https://github.com/googleapis/genai-toolbox/issues/1044)) ([d91bdfc](https://github.com/googleapis/genai-toolbox/commit/d91bdfcbdcbf5fcae6e17770c88c5ffba4115d67))


### Bug Fixes

* Correct the capitalization of `map` manifests ([#1139](https://github.com/googleapis/genai-toolbox/issues/1139)) ([0b0457c](https://github.com/googleapis/genai-toolbox/commit/0b0457c8e6b78f53a2f1929c05d46fb31421fbca))
* Remove unnecessary fields from `map` parameter manifests ([#1138](https://github.com/googleapis/genai-toolbox/issues/1138)) ([fbe8c1a](https://github.com/googleapis/genai-toolbox/commit/fbe8c1a9c0f28797443bf9cb32d63bfbc1072881))
* **tools/looker:** Add authorized invocation feature to all Looker tools ([#1091](https://github.com/googleapis/genai-toolbox/issues/1091)) ([3b1cce7](https://github.com/googleapis/genai-toolbox/commit/3b1cce72e7ff4f6b3a0a31db0564dc45b8302caa))
* Update ui info log to reflect port ([#1125](https://github.com/googleapis/genai-toolbox/issues/1125)) ([6d691d5](https://github.com/googleapis/genai-toolbox/commit/6d691d582f18137de504d39f372c5104b7392bff))

## [0.11.0](https://github.com/googleapis/genai-toolbox/compare/v0.11.0...v0.11.0) (2025-08-05)


### ⚠ BREAKING CHANGES

* **tools/bigquery-sql:** Ensure invoke always returns a non-null value ([#1020](https://github.com/googleapis/genai-toolbox/issues/1020)) ([9af55b6](https://github.com/googleapis/genai-toolbox/commit/9af55b651d836f268eda342ea27380e7c9967c94))
* **tools/bigquery-execute-sql:** Update the return messages ([#1034](https://github.com/googleapis/genai-toolbox/issues/1034)) ([051e686](https://github.com/googleapis/genai-toolbox/commit/051e686476a781ca49f7617764d507916a1188b8))

### Features

* Add TiDB source and tool ([#829](https://github.com/googleapis/genai-toolbox/issues/829)) ([6eaf36a](https://github.com/googleapis/genai-toolbox/commit/6eaf36ac8505d523fa4f5a4ac3c97209fd688cef))
* Interactive web UI for Toolbox  ([#1065](https://github.com/googleapis/genai-toolbox/issues/1065)) ([8749b03](https://github.com/googleapis/genai-toolbox/commit/8749b030035e65361047c4ead13dfacb8e9a9b59))
* **prebuiltconfigs/cloud-sql-postgres:** Introduce additional parameter to limit context in list tables ([#1062](https://github.com/googleapis/genai-toolbox/issues/1062)) ([c3a58e1](https://github.com/googleapis/genai-toolbox/commit/c3a58e1d1678dc14d8de5006511df597fd75faa3))
* **tools/looker-query-url:** Add support for `looker-query-url` tool ([#1015](https://github.com/googleapis/genai-toolbox/issues/1015)) ([327ddf0](https://github.com/googleapis/genai-toolbox/commit/327ddf0439058aa5ecd2c7ae8251fcde6aeff18c))
* **tools/dataplex-lookup-entry:** Add support for `dataplex-lookup-entry` tool ([#1009](https://github.com/googleapis/genai-toolbox/issues/1009)) ([5fa1660](https://github.com/googleapis/genai-toolbox/commit/5fa1660fc8631989b4d13abea205b6426bb506a5))

### Bug Fixes

* **tools/bigquery,mssql,mysql,postgres,spanner,tidb:** Add query logging to execute-sql tools ([#1069](https://github.com/googleapis/genai-toolbox/issues/1069)) ([0527532]([https://github.com/googleapis/genai-toolbox/commit/0527532bd7085ef9eb8f9c30f430a2f2f35cef32))

## [0.10.0](https://github.com/googleapis/genai-toolbox/compare/v0.9.0...v0.10.0) (2025-07-25)


### Features

* Add `Map` parameters support ([#928](https://github.com/googleapis/genai-toolbox/issues/928)) ([4468bc9](https://github.com/googleapis/genai-toolbox/commit/4468bc920bbf27dce4ab160197587b7c12fcd20f))
* Add Dataplex source and tool ([#847](https://github.com/googleapis/genai-toolbox/issues/847)) ([30c16a5](https://github.com/googleapis/genai-toolbox/commit/30c16a559e8d49a9a717935269e69b97ec25519a))
* Add Looker source and tool ([#923](https://github.com/googleapis/genai-toolbox/issues/923)) ([c67e01b](https://github.com/googleapis/genai-toolbox/commit/c67e01bcf998e7b884be30ebb1fd277c89ed6ffc))
* Add support for null optional parameter ([#802](https://github.com/googleapis/genai-toolbox/issues/802)) ([a817b12](https://github.com/googleapis/genai-toolbox/commit/a817b120ca5e09ce80eb8d7544ebbe81fc28b082)), closes [#736](https://github.com/googleapis/genai-toolbox/issues/736)
* **prebuilt/alloydb-admin-config:** Add alloydb control plane as a prebuilt config ([#937](https://github.com/googleapis/genai-toolbox/issues/937)) ([0b28b72](https://github.com/googleapis/genai-toolbox/commit/0b28b72aa0ca2cdc87afbddbeb7f4dbb9688593d))
* **prebuilt/mysql,prebuilt/mssql:** Add generic mysql and mssql prebuilt tools ([#983](https://github.com/googleapis/genai-toolbox/issues/983)) ([c600c30](https://github.com/googleapis/genai-toolbox/commit/c600c30374443b6106c1f10b60cd334fd202789b))
* **server/mcp:** Support MCP version 2025-06-18 ([#898](https://github.com/googleapis/genai-toolbox/issues/898)) ([313d3ca](https://github.com/googleapis/genai-toolbox/commit/313d3ca0d084a3a6e7ac9a21a862aa31bf3edadd))
* **sources/mssql:** Add support for encrypt connection parameter ([#874](https://github.com/googleapis/genai-toolbox/issues/874)) ([14a868f](https://github.com/googleapis/genai-toolbox/commit/14a868f2a0780b94c2ca104419b2ff098778303b))
* **sources/firestore:** Add Firestore as Source ([#786](https://github.com/googleapis/genai-toolbox/issues/786)) ([2bb790e](https://github.com/googleapis/genai-toolbox/commit/2bb790e4f8194b677fe0ba40122d409d0e3e687e))
* **sources/mongodb:** Add MongoDB Source ([#969](https://github.com/googleapis/genai-toolbox/issues/969)) ([74dbd61](https://github.com/googleapis/genai-toolbox/commit/74dbd6124daab6192dd880dbd1d15f36861abf74))
* **tools/alloydb-wait-for-operation:** Add wait for operation tool with exponential backoff ([#920](https://github.com/googleapis/genai-toolbox/issues/920)) ([3f6ec29](https://github.com/googleapis/genai-toolbox/commit/3f6ec2944ede18ee02b10157cc048145bdaec87a))
* **tools/mongodb-aggregate:** Add MongoDB `aggregate` Tools ([#977](https://github.com/googleapis/genai-toolbox/issues/977)) ([bd399bb](https://github.com/googleapis/genai-toolbox/commit/bd399bb0fb7134469345ed9a1111ea4209440867))
* **tools/mongodb-delete:** Add MongoDB `delete` Tools ([#974](https://github.com/googleapis/genai-toolbox/issues/974)) ([78e9752](https://github.com/googleapis/genai-toolbox/commit/78e9752f620e065246f3e7b9d37062e492247c8a))
* **tools/mongodb-find:** Add MongoDB `find` Tools ([#970](https://github.com/googleapis/genai-toolbox/issues/970)) ([a747475](https://github.com/googleapis/genai-toolbox/commit/a7474752d8d7ea7af1e80a3c4533d2fd4154d897))
* **tools/mongodb-insert:** Add MongoDB `insert` Tools ([#975](https://github.com/googleapis/genai-toolbox/issues/975)) ([4c63f0c](https://github.com/googleapis/genai-toolbox/commit/4c63f0c1e402817a0c8fec611635e99290308d0e))
* **tools/mongodb-update:** Add MongoDB `update` Tools ([#972](https://github.com/googleapis/genai-toolbox/issues/972)) ([dfde52c](https://github.com/googleapis/genai-toolbox/commit/dfde52ca9a8e25e2f3944f52b4c2e307072b6c37))
* **tools/neo4j-execute-cypher:** Add neo4j-execute-cypher for Neo4j sources ([#946](https://github.com/googleapis/genai-toolbox/issues/946)) ([81d0505](https://github.com/googleapis/genai-toolbox/commit/81d05053b2e08338fd6eabe4849c309064f76b6b))
* **tools/neo4j-schema:** Add neo4j-schema tool ([#978](https://github.com/googleapis/genai-toolbox/issues/978)) ([be7db3d](https://github.com/googleapis/genai-toolbox/commit/be7db3dff263625ce64fdb726e81164996b7a708))
* **tools/wait:** Create wait for tool ([#885](https://github.com/googleapis/genai-toolbox/issues/885)) ([ed5ef4c](https://github.com/googleapis/genai-toolbox/commit/ed5ef4caea10ba1dbc49c0fc0a0d2b91cf341d3b))


### Bug Fixes

* Fix document preview pipeline for forked PRs ([#950](https://github.com/googleapis/genai-toolbox/issues/950)) ([481cc60](https://github.com/googleapis/genai-toolbox/commit/481cc608bae807d9e92497bc8863066916f7ef21))
* **prebuilt/firestore:** Mark database field as required in the firestore prebuilt tools ([#959](https://github.com/googleapis/genai-toolbox/issues/959)) ([15417d4](https://github.com/googleapis/genai-toolbox/commit/15417d4e0c7b173e81edbbeb672e53884d186104))
* **prebuilt/cloud-sql-mssql:** Correct source reference for execute_sql tool in cloud-sql-mssql.yaml prebuilt config ([#938](https://github.com/googleapis/genai-toolbox/issues/938)) ([d16728e](https://github.com/googleapis/genai-toolbox/commit/d16728e5c603eab37700876a6ddacbf709fd5823))
* **prebuilt/cloud-sql-mysql:** Update list_table tool ([#924](https://github.com/googleapis/genai-toolbox/issues/924)) ([2083ba5](https://github.com/googleapis/genai-toolbox/commit/2083ba50483951e9ee6101bb832aa68823cd96a5))
* Replace 'float' with 'number' in McpManifest ([#985](https://github.com/googleapis/genai-toolbox/issues/985)) ([59e23e1](https://github.com/googleapis/genai-toolbox/commit/59e23e17250a516e3931996114f32ac6526a4f8e))
* **server/api:** Add logger to context in tool invoke handler ([#891](https://github.com/googleapis/genai-toolbox/issues/891)) ([8ce311f](https://github.com/googleapis/genai-toolbox/commit/8ce311f256481e8f11ecb4aa505b95a562f394ef))
* **sources/looker:** Add agent tag to Looker API calls. ([#966](https://github.com/googleapis/genai-toolbox/issues/966)) ([f55dd6f](https://github.com/googleapis/genai-toolbox/commit/f55dd6fcd099f23bd89df62b268c4a53d16f3bac))
* **tools/bigquery-execute-sql:** Ensure invoke always returns a non-null value ([#925](https://github.com/googleapis/genai-toolbox/issues/925)) ([9a55b80](https://github.com/googleapis/genai-toolbox/commit/9a55b804821a6ccfcd157bcfaee7e599c4a5cb63))
* **tools/mysqlsql:** Unmarshal json data from database during invoke ([#979](https://github.com/googleapis/genai-toolbox/issues/979)) ([ccc3498](https://github.com/googleapis/genai-toolbox/commit/ccc3498cf0a4c43eb909e3850b9e6f582cd48f2a)), closes [#840](https://github.com/googleapis/genai-toolbox/issues/840)

## [0.9.0](https://github.com/googleapis/genai-toolbox/compare/v0.8.0...v0.9.0) (2025-07-11)


### Features

* Dynamic reloading for toolbox config ([#800](https://github.com/googleapis/genai-toolbox/issues/800)) ([4c240ac](https://github.com/googleapis/genai-toolbox/commit/4c240ac3c961cd14738c998ba2d10d5235ef523e))
* **sources/mysql:** Add queryTimeout support to MySQL source ([#830](https://github.com/googleapis/genai-toolbox/issues/830)) ([391cb5b](https://github.com/googleapis/genai-toolbox/commit/391cb5bfe845e554411240a1d9838df5331b25fa))
* **tools/bigquery:** Add optional projectID parameter to bigquery tools ([#799](https://github.com/googleapis/genai-toolbox/issues/799)) ([c6ab74c](https://github.com/googleapis/genai-toolbox/commit/c6ab74c5dad53a0e7885a18438ab3be36b9b7cb3))


### Bug Fixes

* Cleanup unassigned err log ([#857](https://github.com/googleapis/genai-toolbox/issues/857)) ([c081ace](https://github.com/googleapis/genai-toolbox/commit/c081ace46bb24cb3fd2adb21d519489be0d3f3c3))
* Fix docs preview deployment pipeline ([#787](https://github.com/googleapis/genai-toolbox/issues/787)) ([0a93b04](https://github.com/googleapis/genai-toolbox/commit/0a93b0482c8d3c64b324e67408d408f5576ecaf3))
* **tools:** Nil parameter error when arrays are used ([#801](https://github.com/googleapis/genai-toolbox/issues/801)) ([2bdcc08](https://github.com/googleapis/genai-toolbox/commit/2bdcc0841ab37d18e2f0d6fe63fb6f10da3e302b))
* Trigger reload on additional fsnotify operations ([#854](https://github.com/googleapis/genai-toolbox/issues/854)) ([aa8dbec](https://github.com/googleapis/genai-toolbox/commit/aa8dbec97095cf0d7ac771c8084a84e2d3d8ce4e))

## [0.8.0](https://github.com/googleapis/genai-toolbox/compare/v0.7.0...v0.8.0) (2025-07-02)


### ⚠ BREAKING CHANGES

* **postgres,mssql,cloudsqlmssql:** encode source connection url for  sources ([#727](https://github.com/googleapis/genai-toolbox/issues/727))

### Features

* Add support for multiple YAML configuration files ([#760](https://github.com/googleapis/genai-toolbox/issues/760)) ([40679d7](https://github.com/googleapis/genai-toolbox/commit/40679d700eded50d19569923e2a71c51e907a8bf))
* Add support for optional parameters ([#617](https://github.com/googleapis/genai-toolbox/issues/617)) ([4827771](https://github.com/googleapis/genai-toolbox/commit/4827771b78dee9a1284a898b749509b472061527)), closes [#475](https://github.com/googleapis/genai-toolbox/issues/475)
* **mcp:** Support MCP version 2025-03-26 ([#755](https://github.com/googleapis/genai-toolbox/issues/755)) ([474df57](https://github.com/googleapis/genai-toolbox/commit/474df57d62de683079f8d12c31db53396a545fd1))
* **sources/http:** Support disable SSL verification for HTTP Source ([#674](https://github.com/googleapis/genai-toolbox/issues/674)) ([4055b0c](https://github.com/googleapis/genai-toolbox/commit/4055b0c3569c527560d7ad34262963b3dd4e282d))
* **tools/bigquery:** Add templateParameters field for bigquery ([#699](https://github.com/googleapis/genai-toolbox/issues/699)) ([f5f771b](https://github.com/googleapis/genai-toolbox/commit/f5f771b0f3d159630ff602ff55c6c66b61981446))
* **tools/bigtable:** Add templateParameters field for bigtable ([#692](https://github.com/googleapis/genai-toolbox/issues/692)) ([1c06771](https://github.com/googleapis/genai-toolbox/commit/1c067715fac06479eb0060d7067b73dba099ed92))
* **tools/couchbase:** Add templateParameters field for couchbase ([#723](https://github.com/googleapis/genai-toolbox/issues/723)) ([9197186](https://github.com/googleapis/genai-toolbox/commit/9197186b8bea1ac4ec1b39c9c5c110807c8b2ba9))
* **tools/http:** Add support for HTTP Tool pathParams ([#726](https://github.com/googleapis/genai-toolbox/issues/726)) ([fd300dc](https://github.com/googleapis/genai-toolbox/commit/fd300dc606d88bf9f7bba689e2cee4e3565537dd))
* **tools/redis:** Add Redis Source and Tool ([#519](https://github.com/googleapis/genai-toolbox/issues/519)) ([f0aef29](https://github.com/googleapis/genai-toolbox/commit/f0aef29b0c2563e2a00277fbe2784f39f16d2835))
* **tools/spanner:** Add templateParameters field for spanner ([#691](https://github.com/googleapis/genai-toolbox/issues/691)) ([075dfa4](https://github.com/googleapis/genai-toolbox/commit/075dfa47e1fd92be4847bd0aec63296146b66455))
* **tools/sqlitesql:** Add templateParameters field for sqlitesql ([#687](https://github.com/googleapis/genai-toolbox/issues/687)) ([75e254c](https://github.com/googleapis/genai-toolbox/commit/75e254c0a4ce690ca5fa4d1741550ce54734b226))
* **tools/valkey:** Add Valkey Source and Tool ([#532](https://github.com/googleapis/genai-toolbox/issues/532)) ([054ec19](https://github.com/googleapis/genai-toolbox/commit/054ec198b97ba9f36f67dd12b2eff0cc6bc4d080))


### Bug Fixes

* **bigquery,mssql:** Fix panic on tools with array param ([#722](https://github.com/googleapis/genai-toolbox/issues/722)) ([7a6644c](https://github.com/googleapis/genai-toolbox/commit/7a6644cf0c5413e5c803955c88a2cfd0a2233ed3))
* **postgres,mssql,cloudsqlmssql:** Encode source connection url for  sources ([#727](https://github.com/googleapis/genai-toolbox/issues/727)) ([67964d9](https://github.com/googleapis/genai-toolbox/commit/67964d939f27320b63b5759f4b3f3fdaa0c76fbf)), closes [#717](https://github.com/googleapis/genai-toolbox/issues/717)
* Set default value to field's type during unmarshalling ([#774](https://github.com/googleapis/genai-toolbox/issues/774)) ([fafed24](https://github.com/googleapis/genai-toolbox/commit/fafed2485839cf1acc1350e8a24103d2e6356ee0)), closes [#771](https://github.com/googleapis/genai-toolbox/issues/771)
* **server/mcp:** Do not listen from port for stdio ([#719](https://github.com/googleapis/genai-toolbox/issues/719)) ([d51dbc7](https://github.com/googleapis/genai-toolbox/commit/d51dbc759ba493021d3ec6f5417fc04c21f7044f)), closes [#711](https://github.com/googleapis/genai-toolbox/issues/711)
* **tools/mysqlexecutesql:** Handle nil panic and connection leak in Invoke ([#757](https://github.com/googleapis/genai-toolbox/issues/757)) ([7badba4](https://github.com/googleapis/genai-toolbox/commit/7badba42eefb34252be77b852a57d6bd78dd267d))
* **tools/mysqlsql:** Handle nil panic and connection leak in invoke ([#758](https://github.com/googleapis/genai-toolbox/issues/758)) ([cbb4a33](https://github.com/googleapis/genai-toolbox/commit/cbb4a333517313744800d148840312e56340f3fd))

## [0.7.0](https://github.com/googleapis/genai-toolbox/compare/v0.6.0...v0.7.0) (2025-06-10)


### Features

* Add templateParameters field for mssqlsql ([#671](https://github.com/googleapis/genai-toolbox/issues/671)) ([b81fc6a](https://github.com/googleapis/genai-toolbox/commit/b81fc6aa6ccdfbc15676fee4d87041d9ad9682fa))
* Add templateParameters field for mysqlsql ([#663](https://github.com/googleapis/genai-toolbox/issues/663)) ([0a08d2c](https://github.com/googleapis/genai-toolbox/commit/0a08d2c15dcbec18bb556f4dc49792ba0c69db46))
* **metrics:** Add user agent for prebuilt tools ([#669](https://github.com/googleapis/genai-toolbox/issues/669)) ([29aa0a7](https://github.com/googleapis/genai-toolbox/commit/29aa0a70da3c2eb409a38993b3782da8bec7cb85))
* **tools/postgressql:** Add templateParameters field ([#615](https://github.com/googleapis/genai-toolbox/issues/615)) ([b763469](https://github.com/googleapis/genai-toolbox/commit/b76346993f298b4f7493a51405d0a287bacce05f))


### Bug Fixes

* Improve versionString ([#658](https://github.com/googleapis/genai-toolbox/issues/658)) ([cf96f4c](https://github.com/googleapis/genai-toolbox/commit/cf96f4c249f0692e3eb19fc56c794ca6a3079307))
* **server/stdio:** Notifications should not return a response ([#638](https://github.com/googleapis/genai-toolbox/issues/638)) ([69d047a](https://github.com/googleapis/genai-toolbox/commit/69d047af46f1ec00f236db8a978a7a7627217fd2))
* **tools/mysqlsql:** Handled the null value for string case in mysqlsql tools ([#641](https://github.com/googleapis/genai-toolbox/issues/641)) ([ef94648](https://github.com/googleapis/genai-toolbox/commit/ef94648455c3b20adda4f8cff47e70ddccac8c06))
* Update path library ([#678](https://github.com/googleapis/genai-toolbox/issues/678)) ([4998f82](https://github.com/googleapis/genai-toolbox/commit/4998f8285287b5daddd0043540f2cf871e256db5)), closes [#662](https://github.com/googleapis/genai-toolbox/issues/662)

## [0.6.0](https://github.com/googleapis/genai-toolbox/compare/v0.5.0...v0.6.0) (2025-05-28)


### Features

* Add Execute sql tool for SQL Server(MSSQL) ([#585](https://github.com/googleapis/genai-toolbox/issues/585)) ([6083a22](https://github.com/googleapis/genai-toolbox/commit/6083a224aa650caf4e132b4a704323c5f18c4986))
* Add mysql-execute-sql tool ([#577](https://github.com/googleapis/genai-toolbox/issues/577)) ([8590061](https://github.com/googleapis/genai-toolbox/commit/8590061ae4908da0e4b1bd6f7cf7ee8d972fa5ba))
* Add new BigQuery tools: execute_sql, list_datatset_ids, list_table_ids, get_dataset_info, get_table_info ([0fd88b5](https://github.com/googleapis/genai-toolbox/commit/0fd88b574b4ab0d3bee4585999b814675d3b74ed))
* Add spanner-execute-sql tool ([#576](https://github.com/googleapis/genai-toolbox/issues/576)) ([d65747a](https://github.com/googleapis/genai-toolbox/commit/d65747a2dcf3022f22c86a1524ee28c8229f7c20))
* Add support for read-only in Spanner tool ([#563](https://github.com/googleapis/genai-toolbox/issues/563)) ([6512704](https://github.com/googleapis/genai-toolbox/commit/6512704e77088d92fea53a85c6e6cbf4b99c988d))
* Adding support for the --prebuilt flag ([#604](https://github.com/googleapis/genai-toolbox/issues/604)) ([a29c800](https://github.com/googleapis/genai-toolbox/commit/a29c80012eec4729187c12968b53051d20b263a7))
* Support MCP stdio transport protocol ([#607](https://github.com/googleapis/genai-toolbox/issues/607)) ([1702ce1](https://github.com/googleapis/genai-toolbox/commit/1702ce1e00a52170a4271ac999caf534ba00196f))


### Bug Fixes

* Explicitly set query location for BigQuery queries ([#586](https://github.com/googleapis/genai-toolbox/issues/586)) ([eb52b66](https://github.com/googleapis/genai-toolbox/commit/eb52b66d82aaa11be6b1489335f49cba8168099b))
* Fix spellings in comments ([#561](https://github.com/googleapis/genai-toolbox/issues/561)) ([b58bf76](https://github.com/googleapis/genai-toolbox/commit/b58bf76ddaba407e3fd995dfe86d00a09484e14a))
* Prevent tool calls through MCP when auth is required ([#544](https://github.com/googleapis/genai-toolbox/issues/544)) ([e747b6e](https://github.com/googleapis/genai-toolbox/commit/e747b6e289730c17f68be8dec0c6fa6021bb23bd))
* Reinitialize required slice if nil ([#571](https://github.com/googleapis/genai-toolbox/issues/571)) ([04dcf47](https://github.com/googleapis/genai-toolbox/commit/04dcf4791272e1dd034b9a03664dd8dbe77fdddd)), closes [#564](https://github.com/googleapis/genai-toolbox/issues/564)

## [0.5.0](https://github.com/googleapis/genai-toolbox/compare/v0.4.0...v0.5.0) (2025-05-06)


### Features

* Add Couchbase as Source and Tool ([#307](https://github.com/googleapis/genai-toolbox/issues/307)) ([d7390b0](https://github.com/googleapis/genai-toolbox/commit/d7390b06b7bcb15411388e9a4dbcfe75afcca1ee))
* Add postgres-execute-sql tool ([#490](https://github.com/googleapis/genai-toolbox/issues/490)) ([11ea7bc](https://github.com/googleapis/genai-toolbox/commit/11ea7bc584aa4ca8e8b0e7a355f6666ccbea2883))


## [0.4.0](https://github.com/googleapis/genai-toolbox/compare/v0.3.0...v0.4.0) (2025-04-23)


### Features

* Add `AuthRequired` to Neo4j & Dgraph Tools ([#434](https://github.com/googleapis/genai-toolbox/issues/434)) ([afbf4b2](https://github.com/googleapis/genai-toolbox/commit/afbf4b2daeb01119a22ce18469bffb9e9f57d2f8))
* Add `AuthRequired` to tool manifest ([#433](https://github.com/googleapis/genai-toolbox/issues/433)) ([d9388ad](https://github.com/googleapis/genai-toolbox/commit/d9388ad57e832570aab56b9b357c1fb0ba994852))
* Add BigQuery source and tool ([#463](https://github.com/googleapis/genai-toolbox/issues/463)) ([8055aa5](https://github.com/googleapis/genai-toolbox/commit/8055aa519fe6e7993ba524f8f7e684fbfdecc1b9))
* Add Bigtable source and tool ([#418](https://github.com/googleapis/genai-toolbox/issues/418)) ([ae53b8e](https://github.com/googleapis/genai-toolbox/commit/ae53b8eeff9d0e9ec14d9c6d4286c856cc8f1811))
* Add IAM AuthN to AlloyDB Source ([#399](https://github.com/googleapis/genai-toolbox/issues/399)) ([e8ed447](https://github.com/googleapis/genai-toolbox/commit/e8ed447d9153c60a1d6321285587e6e4ca930f87))
* Add IAM AuthN to Cloud SQL Sources ([#414](https://github.com/googleapis/genai-toolbox/issues/414)) ([be85b82](https://github.com/googleapis/genai-toolbox/commit/be85b820785dbce79133b0cf8788bde75ff25fee))
* Add toolset feature to mcp ([#425](https://github.com/googleapis/genai-toolbox/issues/425)) ([e307857](https://github.com/googleapis/genai-toolbox/commit/e307857085ac4c8c2ee8292c914daa5534ba74bf)), closes [#403](https://github.com/googleapis/genai-toolbox/issues/403)
* Add SQLite source and tool ([#438](https://github.com/googleapis/genai-toolbox/issues/438)) ([fc14cbf](https://github.com/googleapis/genai-toolbox/commit/fc14cbfd078d465591e4fefb80542759e82a2731))
* Support env replacement for tools.yaml ([#462](https://github.com/googleapis/genai-toolbox/issues/462)) ([eadb678](https://github.com/googleapis/genai-toolbox/commit/eadb678a7bd4ce74a3b1160f5ed8966ffbb13c61))


### Bug Fixes

* [#419](https://github.com/googleapis/genai-toolbox/issues/419) TLS https URL for SSE endpoint ([#420](https://github.com/googleapis/genai-toolbox/issues/420)) ([0a7d3ff](https://github.com/googleapis/genai-toolbox/commit/0a7d3ff06b88051c752b6d53bc964ed6e6be400e))
* **docs:** Fix link 'Edit this page' ([#454](https://github.com/googleapis/genai-toolbox/issues/454)) ([969065e](https://github.com/googleapis/genai-toolbox/commit/969065e561f28ddb9755d99bbe0b288040198296)), closes [#427](https://github.com/googleapis/genai-toolbox/issues/427)
* Update http error code from invocation ([#468](https://github.com/googleapis/genai-toolbox/issues/468)) ([ff7c0ff](https://github.com/googleapis/genai-toolbox/commit/ff7c0ffc65172a335e8d3321e5a6b92d38dc7e6d)), closes [#465](https://github.com/googleapis/genai-toolbox/issues/465)

## [0.3.0](https://github.com/googleapis/genai-toolbox/compare/v0.2.1...v0.3.0) (2025-04-04)


### Features

* Add 'alloydb-ai-nl' tool ([#358](https://github.com/googleapis/genai-toolbox/issues/358)) ([f02885f](https://github.com/googleapis/genai-toolbox/commit/f02885fd4a919103fdabaa4ca38d975dc8497542))
* Add HTTP Source and Tool ([#332](https://github.com/googleapis/genai-toolbox/issues/332)) ([64da5b4](https://github.com/googleapis/genai-toolbox/commit/64da5b4efe7d948ceb366c37fdaabd42405bc932))
* Adding support for Model Context Protocol (MCP). ([#396](https://github.com/googleapis/genai-toolbox/issues/396)) ([a7d1d4e](https://github.com/googleapis/genai-toolbox/commit/a7d1d4eb2ae337b463d1b25ccb25c3c0eb30df6f))
* Added [toolbox-core](https://pypi.org/project/toolbox-core/) SDK – easily integrate Toolbox into any Python function calling framework


### Bug Fixes

* Add `tools-file` flag and deprecate `tools_file`  ([#384](https://github.com/googleapis/genai-toolbox/issues/384)) ([34a7263](https://github.com/googleapis/genai-toolbox/commit/34a7263fdce40715de20ef5677f94be29f9f5c98)), closes [#383](https://github.com/googleapis/genai-toolbox/issues/383)

## [0.2.1](https://github.com/googleapis/genai-toolbox/compare/v0.2.0...v0.2.1) (2025-03-20)


### Bug Fixes

* Fix variable name in quickstart ([#336](https://github.com/googleapis/genai-toolbox/issues/336)) ([5400127](https://github.com/googleapis/genai-toolbox/commit/54001278878042aff75ed421b9fbe70008e9dd4d))
* **source/alloydb:** Correct user agents not being sent ([#323](https://github.com/googleapis/genai-toolbox/issues/323)) ([ce12a34](https://github.com/googleapis/genai-toolbox/commit/ce12a344ed6290c7c6e36ee117318c20d6fdccc2))

## [0.2.0](https://github.com/googleapis/genai-toolbox/compare/v0.1.0...v0.2.0) (2025-03-03)


### ⚠ BREAKING CHANGES

* Rename "AuthSource" in favor of "AuthService" ([#297](https://github.com/googleapis/genai-toolbox/issues/297))

### Features

* Rename "AuthSource" in favor of "AuthService" ([#297](https://github.com/googleapis/genai-toolbox/issues/297)) ([04cb5fb](https://github.com/googleapis/genai-toolbox/commit/04cb5fbc3e1876d1cf83d3f3de2c176ee2862d63))


### Bug Fixes

* Add items to parameter manifest ([#293](https://github.com/googleapis/genai-toolbox/issues/293)) ([541612d](https://github.com/googleapis/genai-toolbox/commit/541612d72d0123b285bb9f58c9cf1bfd61ebd902))
* **source/cloud-sql:** Correct user agents not being sent ([#306](https://github.com/googleapis/genai-toolbox/issues/306)) ([584c8ae](https://github.com/googleapis/genai-toolbox/commit/584c8aea438eeb991935b4347c2c3b2cb7144cbf))
* Throw error when items field is missing from array parameter ([#296](https://github.com/googleapis/genai-toolbox/issues/296)) ([9193836](https://github.com/googleapis/genai-toolbox/commit/9193836effaae79204f73a8c5d26668a95d2cb91))
* Validate required common fields for parameters ([#298](https://github.com/googleapis/genai-toolbox/issues/298)) ([e494d11](https://github.com/googleapis/genai-toolbox/commit/e494d11e6e1651138dcd527171f63d4fa8604211))


### Miscellaneous Chores

* Release 0.2.0 ([#314](https://github.com/googleapis/genai-toolbox/issues/314)) ([d7ccf73](https://github.com/googleapis/genai-toolbox/commit/d7ccf730e7c0c752615f8a7ea162836c5f9950da))

## [0.1.0](https://github.com/googleapis/genai-toolbox/compare/v0.0.5...v0.1.0) (2025-02-06)


### ⚠ BREAKING CHANGES

* **langchain-sdk:** The SDK for `toolbox-langchain` is now located [here](https://github.com/googleapis/genai-toolbox-langchain-python).

### Features

* Add Cloud SQL for SQL Server Source and Tool ([#223](https://github.com/googleapis/genai-toolbox/issues/223)) ([9bad952](https://github.com/googleapis/genai-toolbox/commit/9bad9520604aa363a6d73f5ce14686895c2f4333))
* Add Cloud SQL for MySQL Source and Tool ([#221](https://github.com/googleapis/genai-toolbox/issues/221)) ([f1f61d7](https://github.com/googleapis/genai-toolbox/commit/f1f61d70877a1c7cc9080f6d70112bd0c0533473))
* Add Dgraph Source and Tool ([#233](https://github.com/googleapis/genai-toolbox/issues/233)) ([617cc87](https://github.com/googleapis/genai-toolbox/commit/617cc872d1d692138a712d39fb7c1a405e9c1876))
* Add local quickstart ([#232](https://github.com/googleapis/genai-toolbox/issues/232)) ([497fb06](https://github.com/googleapis/genai-toolbox/commit/497fb06fae6d04adaad11fa78eb04282d0225dbd))
* Add user agents for cloud sources ([#244](https://github.com/googleapis/genai-toolbox/issues/244)) ([8452f8e](https://github.com/googleapis/genai-toolbox/commit/8452f8eb4457dcb0e360a9d9ae5b6e14e78806b1))
* Add MySQL Source ([#250](https://github.com/googleapis/genai-toolbox/issues/250)) ([378692a](https://github.com/googleapis/genai-toolbox/commit/378692ab50a90dcc1c3353052d0741cfd318c79d))
* Add MSSQL source ([#255](https://github.com/googleapis/genai-toolbox/issues/255)) ([8fca0a9](https://github.com/googleapis/genai-toolbox/commit/8fca0a95ee5e79e30919b05592af643ba57f3183))


### Bug Fixes

* Auth token verification failure should not throw error immediately ([#234](https://github.com/googleapis/genai-toolbox/issues/234)) ([4639cc6](https://github.com/googleapis/genai-toolbox/commit/4639cc6560f09b6b8203650ccce424ce59aa0c14))
* Fix typo in postgres test ([#216](https://github.com/googleapis/genai-toolbox/issues/216)) ([0c3d12a](https://github.com/googleapis/genai-toolbox/commit/0c3d12ae04a752fddcff06e92967910cdd643bbf))
* **mssql:** Fix mssql tool kind to mssql-sql ([#249](https://github.com/googleapis/genai-toolbox/issues/249)) ([1357be2](https://github.com/googleapis/genai-toolbox/commit/1357be2569b5f8d31b2b72fa83749fa8519fc8bd))
* **mysql:** Fix mysql tool kind to mysql-sql ([#248](https://github.com/googleapis/genai-toolbox/issues/248)) ([669d6b7](https://github.com/googleapis/genai-toolbox/commit/669d6b7239c36f612f02948716cf167c5a2eaa10))
* Schema float type ([#264](https://github.com/googleapis/genai-toolbox/issues/264)) ([1702f74](https://github.com/googleapis/genai-toolbox/commit/1702f74e9937eb4539c38c7152fe474870e61591))
* Typos at test cases ([#265](https://github.com/googleapis/genai-toolbox/issues/265)) ([b7c5661](https://github.com/googleapis/genai-toolbox/commit/b7c5661215c431c8590a60e029f3c340132574b7))
* Update README and quickstart with the correct async APIs. ([#269](https://github.com/googleapis/genai-toolbox/issues/269)) ([21eef2e](https://github.com/googleapis/genai-toolbox/commit/21eef2e198683d2f7fd0e606a4410b4f3a51686e))
* Update tool invoke to return json ([#266](https://github.com/googleapis/genai-toolbox/issues/266)) ([ad58cd5](https://github.com/googleapis/genai-toolbox/commit/ad58cd5855be9e1b73926e16527fb89ce778b8d9))

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
