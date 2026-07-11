This release consists primarily of internal changes: the local database can now be encrypted, the database design has been reworked and there is a strict separation between reads and writes.

Encryption is implemented using a fork of [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) which already had some incomplete changes to implement [SQLCipher](https://github.com/sqlcipher/sqlcipher). It is largely based on the work done here https://github.com/mattn/go-sqlite3/pull/1109. Changes required for encryption has been merged with changes in the latest release, the patches are available in `sidecar/third_party/go-sqlite3/UPSTREAM-CHANGES.patch`.

# Changelog

All notable changes to this project will be documented in this file.

## Unreleased

[cc9c169](cc9c169736001a1c3b200fc5e2e2330ebc1e8891)...[d7c0620](d7c062091581d55fa78065a026453146847c0f2e)

### Documentation

- Concurrent writes and constructing timeline ([79251c7](79251c7f0e01859be71fd1fb5a53b7da37fc3fc7))
- Todo plugins ([8c680f8](8c680f8d1beef84f310402690344b4b0ad4bdcbc))
- Todo encryption ([733be29](733be29a43e1038b6e11be1602ea10d9ee19ce87))

### Features

- Split read and write queries (#22) ([b2e4fc2](b2e4fc2d4a7e6753abdd7cc1e5698812c5f9842a))
- Rework db module  (#25) ([9d605a1](9d605a17d281faba97f3540e1ee7e51e313193a3))
- Rework db design (#26) ([3be0857](3be08571242f9f8d38c6d9d2c93cae69b2960408))

### Miscellaneous Tasks

- Mvp vibe slop - sqlite encryption ([2e3e68b](2e3e68b5bbc5d1f87b7f2008967874b05fdda725))
- Mvp vibe slop - bump mattn/go-sqlite3 master ([d1e0635](d1e06352f0fbbd537b18f168c5a228e966ecd6eb))
- Add status-checks workflow (#23) ([ef641b0](ef641b006fcd2740f21e283a36594e4007756263))
- V0.0.1-15 (#27) ([d7c0620](d7c062091581d55fa78065a026453146847c0f2e))

## 0.0.1-14 - 2026-07-06

[225a5e3](225a5e387fac4f76eb93f72b71a28de0d7c92e34)...[cc9c169](cc9c169736001a1c3b200fc5e2e2330ebc1e8891)

### Features

- Mvp vibe slop - allow exporting in bulk and as a pdf. pdfs are generated from handlebar templates which are rendered in Chrome with tailwind (vendored) available for styling. chrome is lazy loaded and cached so that the installer doesn't bloat ([dc0eeb3](dc0eeb308d72fcf56bd0557863460ccaa69e9636))

### Miscellaneous Tasks

- Mvp vibe slop - fix installer build ([d4e3972](d4e39722b8bd94063a6335b676d44d0638f6dbe4))
- Mvp vibe slop - fix crash after installing, missing runtime libraries in build ([48d7ac0](48d7ac0f65ddd2d6e146395744f1c2feab2d37f3))
- V0.0.1-14 (#21) ([f1fd4d0](f1fd4d0aab0b0f1db2286684d1f20e009019158c))

## 0.0.1-13 - 2026-07-05

[d0df1a3](d0df1a370cca3ad119b8f4d62eb4edc00fd7bb9e)...[225a5e3](225a5e387fac4f76eb93f72b71a28de0d7c92e34)

### Miscellaneous Tasks

- Test release for windows auto updates ([dfd848b](dfd848b118909f3577cf7111b0ad651237b148a7))

## 0.0.1-12 - 2026-07-05

[ffa4b19](ffa4b19fc15f64fe1737d8318f5f67213e7db86f)...[d0df1a3](d0df1a370cca3ad119b8f4d62eb4edc00fd7bb9e)

### Miscellaneous Tasks

- Mvp vibe slop - update according to release channel, should've thought of this before ([84b17b9](84b17b9db3235d9ef05e6f444842276daf22b27d))
- Mvp vibe slop - upload artefacts for ci build runs ([73b4bd3](73b4bd327474702394a821b1dabd853922c0df37))
- Mvp vibe slop - fix ci tags, marking it as development with version at 0 should allow users to update to either a canary, staging or stable release since these will always be bigger. previously it was 1 (2 after normalizing because major has to start at 1) ([54ae748](54ae748627fcccb6125f82ea188193cf32e66748))
- Mvp vibe slop - in app updates causing data loss ([1e2ba51](1e2ba5172c08bbb98290ecd11d1cc45efd27b96c))
- Mvp vibe slop - fix update checks by normalizing semver. we kind of need a semver string to map to a number which we can compare to determine if we need to upgrade ([8e8d873](8e8d873b0c6e075e8a160c25ae123b5d4f5a21dd))
- Mvp vibe slop - allow update dialogs to be dismissed ([051400a](051400ac90c7920c70cf566b0ab081b561345891))
- Mvp vibe slop - fix wrong version launched after installing update ([24a40e2](24a40e2dcd9a6365931dd917a4a0886f50f01f92))
- Mvp vibe slop - build app exe's as well as installer exe's. update in place and relaunch ([f27e63f](f27e63fab2914a3d8f58919e8c7c35543bfc956a))

## 0.0.1-11 - 2026-07-05

[0603d71](0603d718099c1d36f735fbad559f586090e6b856)...[ffa4b19](ffa4b19fc15f64fe1737d8318f5f67213e7db86f)

### Miscellaneous Tasks

- Mvp vibe slop - include arch to filenames. it's not tested on windows but works on mac. need to setup windows for dev properly. not supporting linux yet because of wayland which is now default on debian, ubuntu, fedora, whether we can do usage tracking depends on the compositor (gnome, kde, xfce, etc). querying the foreground app is not part of the wayland spec ([c946266](c9462661fd57dc779334229133e13f07526587bc))

## 0.0.1-10 - 2026-07-04

[10062b0](10062b009db608eee97ede13f58202383e6be005)...[0603d71](0603d718099c1d36f735fbad559f586090e6b856)

### Miscellaneous Tasks

- Mvp vibe slop - attempt fix for failing arm64 win build ([3fe99f0](3fe99f050d5b224d1b0145d74849a057e3d0c5dd))
- Mvp vibe slop - update build workflow ([3552152](35521520be8ed3a384b7826020d7845248a828a1))

## 0.0.1-9 - 2026-07-04

[d78a803](d78a803f90eb1e7a46abd9bfa3b41c0910398ba7)...[10062b0](10062b009db608eee97ede13f58202383e6be005)

### Miscellaneous Tasks

- Mvp vibe slop - build arm64 sqlite-vector. new macs on m1 run arm64 windows vm ([5506c53](5506c535e114aba5f7ebf09c860871246492ac27))

## 0.0.1-8 - 2026-07-04

[b7a16b4](b7a16b4fac3b51dc323b38fa9609ddd67a92dadb)...[d78a803](d78a803f90eb1e7a46abd9bfa3b41c0910398ba7)

### Miscellaneous Tasks

- Mvp vibe slop - multiarch ([8864088](8864088b744f5617f83c5cfcbc8b7aed61661bda))
- Mvp vibe slop - fix launch errors after installing ([2ad8220](2ad82208cc812d917de18fdcb249a94f10583e97))
- Mvp vibe slop - update release workflow, macos-13 -> macos-15 ([68649bd](68649bdabd69d72ba252e57f0420d718b111149c))

## 0.0.1-7 - 2026-07-04

[041fc40](041fc409e5ccb9625c3b8c9e66c8c657e38af661)...[b7a16b4](b7a16b4fac3b51dc323b38fa9609ddd67a92dadb)

### Miscellaneous Tasks

- Mvp vibe slop - check for updates ([4332dc8](4332dc85cde3a31145f03523970f4f2b2544247f))

## 0.0.1-6 - 2026-07-04

[68ba300](68ba300f38acec9abde7384192efd32493317df1)...[041fc40](041fc409e5ccb9625c3b8c9e66c8c657e38af661)

### Miscellaneous Tasks

- Authenticate as gh app in releases ([5fc5c8b](5fc5c8b67986aa1f035ba844c5a11e135189b071))
- Update release workflow to use kakak-bot ([4173cf5](4173cf5216f020ed5b1d085b59eba9bcf5da3da1))

## 0.0.1-5 - 2026-07-04

[047bc3e](047bc3e1904f4ca25386ebff0aa8d9a83ca709f3)...[68ba300](68ba300f38acec9abde7384192efd32493317df1)

### Miscellaneous Tasks

- Update release workflow (#7) ([b8ef89c](b8ef89c83221b30e6eb942da2dc47f78a2518558))

## 0.0.1-4 - 2026-07-03

[a6fdb34](a6fdb345cfd4740383d5a75c796fb93429d4d1cb)...[047bc3e](047bc3e1904f4ca25386ebff0aa8d9a83ca709f3)

### Miscellaneous Tasks

- Mvp vibe slop - fix setting secrets on windows, also improve secret handling on windows ([4f41da5](4f41da541e2757461388e7d73f8eefe97e82784d))

## 0.0.1-3 - 2026-07-03

[aa7a2d2](aa7a2d2366fe6c38e5db0e8b2c0988ddc9874ba5)...[a6fdb34](a6fdb345cfd4740383d5a75c796fb93429d4d1cb)

### Miscellaneous Tasks

- Mvp vibe slop - create shortcuts on windows, update title bar config ([84b428a](84b428a95d8e1ba7d91213eff2104a21e941eaaa))

## 0.0.1-2 - 2026-07-03

[da9ccdf](da9ccdffd0d3b75033310a564171c2415593daaa)...[aa7a2d2](aa7a2d2366fe6c38e5db0e8b2c0988ddc9874ba5)

### Miscellaneous Tasks

- Mvp vibe slop - set executable flags ([22c0c92](22c0c92cbda80d958ffa120943dc36d5ebdc4bcf))

## 0.0.1-0 - 2026-07-03

### Bug Fixes

- Pretty logs, use golang-migrate ([64ccdd1](64ccdd106997bbfbf1c9e380598b9a6c69070b88))
- Break apart methods, migrate to nvidia llama nemotron for embedding ([fb2b873](fb2b8730f3020f8f4add6dd1860b0017f81cba78))
- Queue filling up, update transition_event_document_float32_embeddings ([5a113b4](5a113b436f239d07e82a4b08700177ac8149e05d))

### Features

- Add monitor ([2abe034](2abe034ca36ef012684175baff112ecadfd39e6c))
- Add database reporter ([ffde432](ffde4323fd629ae9db5a9d52b7aa52412516823d))
- Add transition reasons ([27e2cb6](27e2cb6bd293d999ad3a7c67dab99349b06d83ae))
- Add transition events to queue ([819303d](819303d17014d5f6d769782960915556b4b6aaef))
- Store embeddings ([ab22618](ab2261833418af69ac490dd78dc26944e062c658))
- Add answerer ([1f80ba1](1f80ba10a14638fda52aab18d705076ed42c3205))
- Migrate to queues because sqlite only allows a single writer, running into errors while writing otherwise ([ad2060d](ad2060d35c52a0c3da19e692881b653807293bd3))

### Miscellaneous Tasks

- Initial commit ([edc9ee4](edc9ee41b77ae99954c43e4385f5d0f6ad1acbd8))
- Go mod init ([d44e33b](d44e33bf07b93f5efac17838881d2151f2f5f35a))
- Setup grpc, worker queue, db ([a2ab00d](a2ab00dd4e0de46c7380614c39dfd17d5a088064))
- Setup sqlc, add Taskfile ([c2415e8](c2415e8f2007f8d2c8fc3c08d1a77e4312e4970c))
- Add kmp app ([175679b](175679bb118f8095f6da3ec2d9b88ed144f2571d))
- Update gitignore ([69110e4](69110e4d24937d25f0250604d4e8e6118ab7e013))
- Mvp vibe slop 1 ([0861762](08617624faee232d71bf9eec3e74e4023fc2fb45))
- Mvp vibe slop 2 ([3e052ac](3e052ac7023fea9ccf19d6f9a6d8b5cc6b0de83b))
- Mvp vibe slop 3 ([984850d](984850d44ddb3a9f7b4ff4b3e8765d3f2157f984))
- Mvp vibe slop fix title bar white strip ([b3bfffc](b3bfffc96cfd5fc8832b1f32fd98ab4d81d61beb))
- Mvp vibe slop 4 ([094eb68](094eb68dee65ceeb024f72e0de5d6e5ef5ed3146))
- Vibe refactor 1 ([ccafb54](ccafb54bd8b33006bbe40c4fcf5f56b74e17e2a7))
- Vibe refactor 2 ([e179603](e17960342b8cede298f11d24faaaa0bb90347908))
- Mvp vibe slop 5 ([e6cf261](e6cf2617351db7a77bac62b95bf7033503efbc30))
- Mvp vibe slop 6 ([f457255](f45725522fee70fa6ba50a02a6430e0386c2528f))
- Mvp vibe slop 7 - projects and entries ([aabb9a9](aabb9a93e1b99be6f68087bb782acd88d545e51e))
- Mvp vibe slop 8 - improve animations ([c44e6ec](c44e6ec132174170178903cecefa3880be3128c4))
- Mvp vibe slop 9 - linear usage event ([2410fde](2410fdebbe2435011bbe49473401931ccf8494e7))
- Mvp vibe slop 10 - animations update ([89bb6dc](89bb6dc85ce4cfdeb56eb2ff08c3b17fd7d971b3))
- Mvp vibe slop 11 - prune & compact ([cc24ae0](cc24ae0e7257d1fb00267ea0d7abe51e4c40ccd1))
- Mvp vibe slop 12 - turboquant ([1e37a18](1e37a18d809c7c71df72c7c95fc134688269645e))
- Mvp vibe slop - usage schedule event fixes ([8b83891](8b838916ec81ccc281fa42a3ff8852cee0924a55))
- Mvp vibe slop - installer ([4ce341d](4ce341d402b9695783fc914f44783ead5ac3c78e))
- Mvp vibe slop - improve ama ([c87c304](c87c304c3bd718ae1241856b691ed0931b4fec4d))
- Mvp vibe slop - landing ([34aa266](34aa266e2ab2b2653b9ca1b58c21f161e996acc5))
- Mvp vibe slop - logos ([bff4361](bff4361b6e03183b0de956873df0e19695027162))
- Mvp vibe slop - landing preview ([91866b6](91866b695c4c3ce416170360c32d28ef7ab425e3))
- Mvp vibe slop - enqueue unindexed events ([9530a70](9530a706526e271b8bcdb6baff1e5705586c602c))
- Update README ([a6427c0](a6427c07355dd0e80bb1ac19449cf24c351cd195))
- Update README ([13bea29](13bea29e7c1bcf3bb50fbe62901dc2c5f31c9464)), Signed-off-by:Naveen Mathew <55116576+nmathew98@users.noreply.github.com>
- Update README ([e328890](e32889007817540588a1b3c9273959964fcd65af)), Signed-off-by:Naveen Mathew <55116576+nmathew98@users.noreply.github.com>
- Mvp vibe slop - mask pw, move save button into appearance card ([3826169](38261697a6f0e47e0ad936c598d766f8e0adef4a))
- Update gitignore ([63e93b3](63e93b36ce40fbfaaec155a67ee1a5cc37305ed2))
- Mvp vibe slop - ci ([e7ee471](e7ee4717223d272a5dcaa22c5f3bc6d0845b9363))
- Mvp vibe slop - o11y ([168b57c](168b57c9b55603d76ab8634e5679238362ddbe49))
- Mvp vibe slop - add gradle wrapper ([bc20165](bc201658e38ad27d08505d3ddf6acbca8f103c60))
- Mvp vibe slop - fix sidecar tests on windows ([43cc9b9](43cc9b9337b6ac3aab38ac2a6bb96f3bd1d319d3))
- Mvp vibe slop - release action ([f0247be](f0247bec3021399b8cbeb1ef91380621093a1cf4))
- Mvp vibe slop - fix installer build ([6013cd3](6013cd32e793599e20149c6302d91e80976e27b9))

### Refactor

- Propagate context ([8d5ea72](8d5ea72ea1a386b15f55749e3c5918afc5482cbc))
- Queue ([13e1b05](13e1b05530e627d1bf821a757b1e66099962c400))

<!-- generated by git-cliff -->
