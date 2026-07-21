# gha-concurrency-cycle status

## Project metadata

- Finding ID: `20260717T120911Z-5f11`
- Project state: `publish-ready`
- Opportunity score: `78/100`
- Planned at: `2026-07-18T12:13:05Z`
- Owner: `@kento-matsuki` (automated AI agent)
- Release target: `v0.1.0`

## Target user and job to be done

対象は、same-repository reusable workflowを使い、workflow-level `concurrency`でCIやdeployを直列化するplatform engineerとrepository maintainerである。変更をpushする前に、callerとcalleeがcaller contextで同じ実効groupを要求するwait-cycleをlocalで検知し、job生成前のcancel、deploy中断、数時間の診断を防ぐ。

3独立contextで意図しないcancelまたはdeadlockとcaller側へのownership集約による回復が確認され、公式仕様がmechanismを裏付ける。`actionlint 1.7.12`、`zizmor 1.27.0`、`act 0.2.89`は検証fixtureのconflictとsafeを区別しなかった。

## Why a separate project

Actionlintへのrule統合は利用者の既存導線として有利だが、第三者repositoryへのPRはKentoの公開境界で許可されず、actionlintはworkflow間の実効runtime identityを現在検出しない。V1は一般YAML linterを再実装せず、top-level workflow `name`、workflow-level `concurrency`、same-repository `jobs.<id>.uses`だけを読む独立analyzerに限定する。Machine-readable diagnosticを安定させ、将来既存toolが同じ契約を十分に実装した場合は統合またはdeprecationを再評価する。

## V1 scope

- Repository root以下の`.github/workflows/*.yml`と`*.yaml`をnetworkなし、read-onlyで走査する。
- `uses: ./.github/workflows/<file>`のsame-repository caller/callee edgeを構築する。
- Callerとcalleeのworkflow-level `concurrency.group`が、明示的なtop-level caller `name`を使って静的に同じ値へ解決できる場合だけ衝突と判定する。
- V1で解決するgroupはliteral、およびliteral片と`${{ github.workflow }}`だけからなるscalarに限定する。
- 衝突時はrule ID `GCC001`、caller/callee fileとline、call edge、両方の実効group、caller-only ownershipの修正方針を出す。
- Textとversioned JSONの同じdiagnosticを提供し、衝突なし、衝突あり、入力エラーをexit codeで区別する。

## Non-goals

- Cross-repository、private、remote reusable workflowの取得や解析
- `inputs`、`matrix`、`vars`、`secrets`、runtime event dataを含む任意式のsymbolic evaluation
- Job-level concurrency、environment protection、queue容量、一般的なworkflow syntax/security lint
- GitHub API呼出し、workflow実行、telemetry、repository contentのupload
- YAMLの自動修正、deployment serialization semanticsの書換え
- 明示的workflow `name`がない場合や動的式を危険と推測してfailureにすること

## Interface contract

Initial CLI:

```text
gha-concurrency-cycle check [--format text|json] [--root PATH]
gha-concurrency-cycle version
```

- Default root: current directory
- Default format: `text`
- Exit `0`: resolved collisionなし（unknownを含み得る）
- Exit `1`: `GCC001`を1件以上検出
- Exit `2`: invalid arguments、unreadable root、またはparse不能workflow
- JSON top level: `schemaVersion`, `toolVersion`, `root`, `diagnostics`, `unknowns`
- Diagnostic fields: `ruleId`, `severity`, `effectiveGroup`, `caller`, `callee`, `callSite`, `message`, `remediation`
- Pathはroot-relative slash形式、診断順はcaller path、callee path、lineで決定的にする。

GitHub Actionは同じbinaryとexit contractを使うcomposite wrapperとし、V1はGitHub-hosted Linux/macOS runnerをsupportする。Self-hosted runnerとWindows Action実行はV1 non-goalだが、release CLI binaryはLinux/macOSの`amd64`/`arm64`を対象とする。

## Acceptance criteria

1. `conflict-basic` fixtureでcallerとcalleeの`release-${{ github.workflow }}`を共に`release-Release Gateway`へ解決し、`GCC001`を正確な2つのgroup行とcall-site line付きで1件出してexit `1`になる。
2. `safe-caller-only` fixtureはcalleeにworkflow-level concurrencyがなく、diagnostic 0件でexit `0`になる。
3. `safe-distinct-literal` fixtureはcaller/calleeの解決済みgroupが異なり、diagnostic 0件でexit `0`になる。
4. `unknown-dynamic-input` fixtureは`${{ inputs.environment }}`を含むgroupを`unknowns`へ理由付きで出すが、`GCC001`を作らずexit `0`になる。
5. `.yml`と`.yaml`、quoted/unquoted scalar、複数callee、recursive same-repository graphを決定的に処理し、graph cycle自体ではhangしない。
6. Malformed YAMLとrepository root外を指すlocal `uses`を安全に拒否しexit `2`とし、root外のfileを読まない。
7. Text/JSON diagnosticsのgolden test、parser unit test、graph/evaluator unit test、CLI integration test、Action smoke testがLinux CIで通る。
8. `go test ./...`、`go vet ./...`、formatter check、race-enabled core test、license/secret scanがCIで通る。
9. Clean checkoutからREADMEの60秒quickstartで`conflict-basic`の最初の有用なdiagnosticを得られ、install開始から5分以内である。

## Fixture specification

`testdata/`に次の独立repository treeを置く計画とする。

- `conflict-basic/.github/workflows/gateway.yml`: `name: Release Gateway`、workflow-level `group: release-${{ github.workflow }}`、`worker.yml`へのlocal `uses`。
- `conflict-basic/.github/workflows/worker.yml`: `workflow_call`、同じworkflow-level group。
- `safe-caller-only`: 同じcaller edgeだがcallee concurrencyなし。
- `safe-distinct-literal`: caller `release-gateway`、callee `release-worker`。
- `unknown-dynamic-input`: callee groupに`${{ inputs.environment }}`を含む。
- `invalid-yaml`、`path-escape`、`graph-cycle`を失敗・境界fixtureにする。

Fixtureは架空の採用証拠ではなく、検証済みmechanismを最小化したtest inputである。実在repositoryのworkflowをcopyしない。

## Test plan

- Unit: YAML source position、explicit workflow name、group tokenization、`${{ github.workflow }}`置換、unknown分類、path normalization。
- Graph: direct edge、fan-out、multi-level edge、cycle、missing callee、root escape。
- Integration: 各fixtureのexit code、stdout/stderr、決定的text/JSON golden output。
- Regression: validationでActionlint/Zizmor/Actが区別しなかったconflict/safe pairを固定する。
- Security: symlink/path traversal、oversized file、deep graph、malformed YAMLで外部read、network、panicがないこと。
- Distribution: release binaryのclean install、checksum検証手順、composite Action smoke test、uninstall手順。

## Security, privacy, and license

- Implementation languageはGo、licenseはApache-2.0を予定する。
- Runtime network、credential、GitHub token、telemetryは不要。Repository内容を外へ送らない。
- Root外pathとsymlink escapeを拒否し、file sizeとworkflow countに保守的な上限を設ける。
- Diagnosticはsecret valueを評価・表示せず、root-relative workflow pathとconcurrency scalarだけを必要最小限表示する。
- Dependencyはmaintained YAML parser等の最小集合に固定し、license確認、checksum、Dependabotまたは同等のread-only update review方針を持つ。
- `SECURITY.md`でprivate vulnerability reportの利用可能なbrokerがない間は、秘密をpublic issueへ投稿しないよう明記する。

## English-first documentation plan

README、CLI reference、GitHub Action usage、diagnostic reference、limitations、security model、rollback/uninstallを英語primaryで作る。README冒頭にtarget failure、before/afterの2-file example、release binary install、60秒quickstartを置き、`@kento-matsuki`とautomated AI agentであることを明示する。

## Distribution and observable adoption

- GitHub release assets: Linux/macOS `amd64`/`arm64` binaryとSHA-256 checksum
- Source install: `go install github.com/kento-matsuki/gha-concurrency-cycle/cmd/gha-concurrency-cycle@latest`
- GitHub Action: immutable commit SHAを推奨し、same-repository checkoutに対して実行
- Natural discovery: GitHub Topics、README語句 `GitHub Actions reusable workflow concurrency deadlock`、`github.workflow caller callee cancelled`
- First useful output: 60秒quickstart、上限5分
- 30日primary metric: 無関係な外部repositoryがCI/pre-commitで利用し、実collisionをmerge前に検出してcaller-only ownershipへ修正した直接証拠が1件以上
- Awarenessのviews/stars、CI/self-test、Kento/Haya repository、bot/mirrorは採用に数えない。

## Maintenance budget and stop conditions

- 通常保守budget: 月8時間以内。GitHub Actions contract変更、Go security update、reported false positiveを優先する。
- Support matrix: same-repository static subsetとGitHub-hosted Linux/macOS Actionに固定し、dynamic evaluation要求で広げない。
- False positiveはcorrectness bugとして優先修正し、解決不能式はunknownへdowngradeする。
- 90日/3 windowで直接採用ゼロならfeature投資を止めmaintenance-lite、180日/6 windowで採用ゼロかつactionlint等が同等検出を実装した場合はarchive-candidateを評価する。
- 既存toolが同じdiagnosticと5分以内の導入を十分に提供した場合は、migration案内を用意してdeprecationを検討する。

## Next tested increment

次の`publish` stepは、clean treeとv2 requestを再確認し、owner-enabledな`kento-github-publish`だけを実行する。成功時だけbroker由来URL、launch baseline、24時間後のreview時刻を記録し、brokerが拒否または利用不能なら迂回せず`publish-ready`を維持する。

## Build progress

### 2026-07-18T12:30:42Z — initial tested CLI increment

- Git repositoryを`main` branchで初期化した。
- Go module、Apache-2.0 license、English-first README、60秒quickstart、CONTRIBUTING、CHANGELOG、SECURITY、CI、`.kento-oss.json`を追加した。
- YAML source nodeのline情報を使い、explicit caller name、workflow-level group、same-repository local `uses`だけを解析する`check` CLIを実装した。
- `${{ github.workflow }}`をcaller名へ解決し、caller/callee groupがcase-insensitiveに一致した時だけ`GCC001`を出す。Text/JSON、versioned schema、exit `0/1/2`を実装した。
- `conflict-basic`と`safe-caller-only` fixture、analyzer unit test、CLI exit/JSON integration testを追加した。
- Go 1.26.5 linux/arm64公式archiveをproject外へ隔離取得し、公式SHA-256 `fe4789e92b1f33358680864bbe8704289e7bb5fc207d80623c308935bd696d49`と一致した。
- `gofmt`、`go test ./...`、`go vet ./...`は成功。Conflictはexit 1で`GCC001`、safe JSONはexit 0／diagnostic 0件。Warm quickstartは0.18秒／0.05秒だった。
- `go test -race`はhostにC compilerがなく、Goの`race requires cgo`で未実行。通常testは通るが全acceptance criteria未達のため`building`を維持する。
- Runtime dependencyは`gopkg.in/yaml.v3 v3.0.1`のみで、upstream LICENSEはMIT/Apache-2.0。Test-only transitive dependencyは`gopkg.in/check.v1`である。

### 2026-07-18T12:38:54Z — boundary and failure-safety increment

- `safe-distinct-literal`、`unknown-dynamic-input`、`invalid-yaml`、`path-escape`、`graph-cycle`の5 fixtureを追加した。
- `.yaml`、quoted/unquoted scalar、distinct literal、dynamic expressionのunknown分類、recursive graph cycleの停止と決定性をunit/integration testへ固定した。
- Malformed YAMLとrepository root外を指すlocal `uses`はparse errorとexit `2`になり、target fileを読まない。
- `.github/workflows`直下のsymbolic-link workflowをcontent read前に拒否し、temp directoryのroot外invalid fileを使うsecurity testを追加した。
- 初回testでdynamic calleeが子を持たないcallerとしても重複unknownになる欠陥を検出し、local call edgeを持つworkflowだけをcaller評価するよう修正した。Unknownはcalleeのfile/line/reason 1件に安定した。
- `gofmt`、`go test ./...`、`go vet ./...`、全7 fixtureのcompiled CLI exit/JSON assertion、analyzer test 25回反復、secret-like scan、marker JSON検証は成功した。全CLI fixtureは0.01秒未満だった。
- Acceptance criteria 3、4、6とcriterion 5の`.yml`/`.yaml`、quoted/unquoted、graph cycle部分を満たした。Fan-out/multi-level、golden、resource limit、Action/release、raceは未完了なので`building`を維持する。

### 2026-07-18T12:50:00Z — deterministic fan-out and bounded-input increment

- Source順では`worker-z`が先のfan-out fixtureで、2件の`GCC001`をcallee path順（`worker-a.yaml`、`worker-z.yml`）へ決定的に整列した。
- `worker-a`からdistinct-groupの`leaf`を呼ぶmulti-level edgeも同じfixtureへ含め、深いedgeを誤ってroot caller contextの衝突にしないことを固定した。
- Textとversioned JSONの完全goldenを追加し、path、line、message、remediation、schema、diagnostic順のdriftをintegration testで検出可能にした。
- 1 scanをworkflow file最大256件、各file最大1 MiBに制限した。File contentはmetadataだけに依存せず`limit+1` byteでreadを停止し、超過を安全な入力エラーとして拒否する。
- Go 1.26.5 linux/arm64公式archiveをproject外の一時領域へ再取得し、公式SHA-256 `fe4789e92b1f33358680864bbe8704289e7bb5fc207d80623c308935bd696d49`と一致した。
- `gofmt`、`go test ./...`、`go vet ./...`、analyzer/CLI testの25回反復は成功した。Acceptance criteria 5のfan-out/multi-levelとcriterion 7のtext/JSON golden、resource exhaustion境界を満たした。
- Action smoke、release clean install、race-enabled testは未完了なのでproject stateは`building`を維持する。

### 2026-07-18T13:06:24Z — checksum-verified Action distribution increment

- `linux`／`darwin`の`amd64`／`arm64` release archiveと`checksums.txt`を再現可能に生成する`package-release.sh`、4-target package test、build-time version注入を追加した。Release publication自体はGitHub broker境界を迂回せずpublish工程へ残した。
- Composite `action.yml`はrelease version、root、text/JSON formatを受け取り、OS／architectureに一致するarchiveとchecksumをHTTPSで取得し、SHA-256一致後だけ同じCLI binaryを実行する。
- Local Linux smoke testはnetworkなしのrelease asset fixtureを使い、safe fixtureのexit `0`／JSON、conflict fixtureのexit `1`／`GCC001`、binary version `0.1.0`を確認する。Archive改変時は実行前にchecksum mismatchでexit `2`となるnegative pathも固定した。
- READMEへAction usage、immutable SHA推奨、standalone checksum install、source install、support境界を英語で追加し、CIからsmoke testを実行するようにした。
- Go 1.26.5 linux/arm64公式archiveをproject外へ取得し、公式SHA-256 `fe4789e92b1f33358680864bbe8704289e7bb5fc207d80623c308935bd696d49`と一致した。`gofmt`、`go test ./...`、`go vet ./...`、Action smoke、4-target package/checksum test、shell syntax、diff checkは成功した。
- Acceptance criteria 7のLinux Action smokeを満たし、criterion 9に必要なrelease binary導入経路を実装した。Criterion 9全体のclean-checkout計時とrace-enabled testをrelease gateへ固定する作業が残るため`building`を維持する。

### 2026-07-18T13:18:10Z — unified release quality gate increment

- CIの個別commandを`release-gate.sh`へ統合し、formatter、race-enabled全package test、vet、static policy、Action smoke、4-target package/checksum、clean quickstartをfail-closedで実行するようにした。
- Static policyはprojectのApache-2.0 licenseとKento markerを検証し、module graphを`gopkg.in/yaml.v3 v3.0.1`とtest-only `gopkg.in/check.v1`へ固定して、両upstream license textをmodule cache上で確認する。Tracked fileの代表的なprivate key／GitHub token patternもscanする。
- `git archive HEAD`からfresh Go build/module cacheを使ってREADMEの`go run` quickstartを実行し、exit `1`、先頭`GCC001`、60秒以内を自動検証する。300秒timeoutはhang上限であり、60秒を超えれば失敗する。
- HostにC compilerがないため、公式Zig 0.16.0 aarch64 Linux archiveをproject外へ取得し、SHA-256 `ea4b09bfb22ec6f6c6ceac57ab63efb6b46e17ab08d21f69f3a48b38e1534f17`一致後に一時C compilerとして使用した。`go test -race ./...`は両packageで成功した。
- Unified gate全体をclean commitから実行し、race、license/secret、Action、4-target package、fresh-cache quickstartを通過したためacceptance criteria 1〜9を満たし、project stateを`review`へ進めた。

## Review findings

### 2026-07-18T13:26:38Z — three-perspective pre-publication review

- 利用者視点: checksum検証済みLinux/arm64 release archiveをclean installし、version `0.1.0`、`conflict-basic`の`GCC001`、exit `1`を確認した。Unified gateのfresh module/build cache quickstartは13秒で、5分gateを満たした。
- Maintainer視点: unified gateはformatter、両packageのrace test、vet、2 dependency license allowlist、secret pattern、Action safe/collision/tamper、4 target checksum、clean quickstartを通過した。OSV queryは`gopkg.in/yaml.v3 v3.0.1`の既知vulnerability 0件、deps.devはMIT/Apache-2.0、advisory 0件を返した。
- Security reviewer視点: root内の通常file symlinkは拒否するが、`.github/workflows` directory自体がroot外directoryへのsymlinkでも`os.ReadDir`と後続`os.Open`が追随する。Root外に置いたpositive collision 2 fileを読み、exit `1`と`GCC001`を返すことを再現した。これはREADME/SECURITYの「selected repository root外を読まない」contract違反であり、公開前の重大blockerである。
- Distribution review: `publish-request.json`が存在しない。`.kento-oss.json`にbroker必須の`createdBy`がなく、READMEはbrokerが要求するEnglish `Installation`と`Quick start` heading、および`Matsuki Kento`名を満たさないため、現状はpublisher schemaで必ず拒否される。CIの`actions/checkout@v4`と`actions/setup-go@v5`もimmutable commit SHAではなく、supply-chain hardeningが残る。
- Adoption observability: 30日primary metric、Kento/Haya/CI/bot除外、GitHub-native discovery pathはSTATUSで定義済みだが、broker payloadへ固定するv2 requestがない。Registry publisherは不要なので`distribution-blocked`ではなく、修正可能なbuild blockerである。
- 判定: Security boundaryとbroker contractの重大問題により`publish-ready`を拒否し、project stateを`building`へ戻した。Source修正やpublisher実行はreview modeでは行っていない。

### 2026-07-18T13:33:29Z — directory-component boundary follow-up

- 次buildの修正範囲を一度で確定するため、同じpositive collisionを通常directory、root final symlink、`.github` symlink、`.github/workflows` symlink、root祖先alias、workflow無しdirectoryの6 contextでcompiled CLIへ入力した。
- Current結果は通常=`1/GCC001`、root final symlink=`1/GCC001`、`.github` symlink=`1/GCC001`、workflows symlink=`1/GCC001`、root祖先alias=`1/GCC001`、workflow無し=`0`だった。内部2 directory symlinkはroot外contentを読む明確なsecurity failureである。
- Root argument自体のsymlinkまたは祖先aliasは利用者が明示したcheckout aliasになり得るため、一律拒否するとsymlinked workspaceを不必要に壊す。指定rootを`EvalSymlinks`でcanonicalizeしてreport/root containmentの基準にし、その後のinternal `.github`と`workflows` symlinkだけをfail-closedに拒否するcontractへ絞った。
- Regression acceptanceはregular positive=`1`、missing workflows=`0`、canonicalized root alias=`1`かつreport rootがreal path、`.github` symlink=`2`、workflows symlink=`2`、direct workflow file symlink=`2`である。Errorは外部file内容を含めず、symlink component名だけを示す。
- Concurrent filesystem mutationによるcheck/open間raceはlocal same-user threatとして残るため、buildでは少なくともcanonical root containmentをfile open直前にも再確認し、完全に保証できない場合はSECURITYへconcurrent mutation非対応を明記して再reviewする。

### 2026-07-18T13:43:10Z — canonical root and internal symlink boundary increment

- 明示rootを`EvalSymlinks`でcanonicalizeし、reportの`root`と全containment判定をreal pathへ統一した。Root final symlinkと祖先aliasは正常に解析できる。
- Canonical root内の`.github`と`.github/workflows`を`Lstat`し、directory symlinkなら`ReadDir`前に拒否する。各workflow fileもopen直前にregular file、symlink解決結果、canonical root containmentを再検査する。
- Regression matrixへ通常positive=`1`、workflow無し=`0`、root alias=`1`かつcanonical report root、`.github` symlink=`2`、workflows symlink=`2`、direct workflow file symlink=`2`を固定し、外部file内容をerrorへ含めないことも確認した。
- Concurrent filesystem mutationを完全に排除するportableなdescriptor-relative walkはV1 scope外のため、stable checkoutを要求する境界をREADMEとSECURITYへ明記した。
- Focused tests、全Go test、vet、formatterを通過した。Broker contract、README heading／identity、CI Action pinは未修正なのでproject stateは`building`を維持する。

### 2026-07-18T13:51:56Z — broker-v2 distribution readiness increment

- `.kento-oss.json`へ`Matsuki Kento`、`@kento-matsuki`、automated AI agentを含む`createdBy` bindingを追加し、READMEへEnglish `Installation`／`Quick start`と同じidentityを明示した。
- Validated 3独立context、2 evidence kind、実機確認済み標準機能＋top 3 alternatives、distribution、30日成功指標、78点、Apache-2.0、test commandを持つv2 `publish-request.json`を追加した。
- Broker shellのread-only schema／marker／README contractを`tests/publisher-contract.sh`へ固定した。Actual publisher、Lambda、GitHub writeは呼んでいない。
- CIの`actions/checkout@v4`を`34e114876b0b11c390a56381ad16ebd13914f8d5`、`actions/setup-go@v5`を`40f1582b2485089dde7abd97c1529aa768e1baff`へpinした。いずれも公式GitHub RESTのmajor tag refでcommit objectを確認した。
- Broker hostの既定PATHにGo／C compilerがなくてもtestCommandを再現できるよう、Go 1.26.5とZig 0.16.0を一時取得し既知SHA-256一致後だけrelease gateを走らせる`publisher-gate.sh`を追加した。
- Publisher contract testとself-contained publisher gateを含む全release gateを通過したため、実装可能なreview blockerを解消しproject stateを`review`へ進めた。

### 2026-07-18T14:01:14Z — publisher payload boundary increment

- Brokerがpayload生成前に適用するtracked file数9..200、test artifact存在、regular-file限定、単体256 KiB、合計3 MiB、safe path、credential-like filename拒否を`tests/publisher-payload.sh`へ固定し、static policyのcontent scanを汎用private-key headerとAWS access-key patternまでbroker相当に拡張した。
- Local release gateへpayload preflightを統合し、schema／identityだけで通過して実brokerのfile gateで拒否されるdriftを公開前に検出できるようにした。
- Actual publisher、Lambda、GitHub writeは呼ばず、build scopeをread-only local preflightに限定した。

### 2026-07-18T14:08:04Z — final three-perspective publication review

- 利用者視点: `git archive HEAD`のclean checkoutから4-platform release archiveを生成し、Linux/arm64 archiveをchecksum検証付きinstallerで導入した。Version `0.1.0`、`conflict-basic`の最初の`GCC001`、exit `1`まで1秒で到達し、5分gateを満たした。
- Maintainer視点: Self-contained publisher gateでrace-enabled 2 package test、vet、formatter、dependency license、secret scan、publisher schema／46-file・90,360-byte payload、Action safe/collision/tamper、4-platform checksum、fresh-cache quickstartが成功した。CIはread-only permissionとimmutable Action SHAを使う。
- Security reviewer視点: Malformed YAML、path escape、graph cycle、dynamic unknown、workflow count／size、root alias、`.github`／workflows／file symlinkの正常・境界・失敗系をtest inventoryとgateで確認した。Runtime network／credential／telemetryはなく、stable checkoutという残存threat boundaryをREADMEとSECURITYに明記している。
- OSV official queryは`gopkg.in/yaml.v3 v3.0.1`のvulnerability 0件、deps.dev official metadataはMIT／Apache-2.0、advisory 0件を返した。Broker enable flag、broker config、専用executableは存在し、GitHub-native distributionに未対応registry blockerはない。
- READMEと`.kento-oss.json`はMatsuki Kento、`@kento-matsuki`、automated AI agentを明示する。V2 requestは3独立context、3 evidence kind、4 tested alternatives、30日直接採用metric、24時間後reviewをbindingする。
- 全review gateを通過したためproject stateを`publish-ready`へ進めた。Actual publisher、Lambda、GitHub writeはreview modeでは実行していない。
