# Changelog

All notable changes to this project will be documented in this file.

## Unreleased

[041fc40](041fc409e5ccb9625c3b8c9e66c8c657e38af661)...[94a3ee8](94a3ee84b0c39f3d2396a7fe54e9109f026a1eb8)

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
