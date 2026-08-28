# 00 — Micropolis 原始碼地圖（已核對）

**推論等級：已確認**（逐檔開啟、列出頂層函式、記錄 SHA-256）。核對日期 2026-08-29。

## 封存身分

| 項目 | 值 |
|---|---|
| repo | <https://github.com/SimHacker/micropolis> |
| commit | `c98f6b08519887b450d9be198bfca5237aab6d0c` |
| commit 日期 | 2026-02-10T21:58:09+01:00 |
| 本地路徑 | `workplace/ref/micropolis/`（gitignore）|
| 規則層路徑 | `micropolis-activity/src/sim/` |
| 授權 | GPL-3.0（**只讀，不照抄**，見 `CLAUDE.md` §7）|

## 被推翻的假說：`s_` ＝ 規則、`w_` ＝ 介面

`CLAUDE.md` §1.3 初版依檔名前綴推測「`s_*.c` 是模擬規則、`w_*.c` 是 X11／Tcl 介面，
`w_` 可以整批跳過」。**這個推測是錯的**，而且錯得很危險——它會讓整個精靈系統
（怪獸、龍捲風、飛機、船、火車、直昇機、爆炸、火災起火點）與整個工具系統
（分區放置、推土機、成本與合法性檢查）被當成介面而跳過。

實際的分界不在前綴，在**內容**：

| 檔案 | 角色 | 為什麼 |
|---|---|---|
| `w_sprite.c` | **規則** | `DoMonsterSprite` `DoTornadoSprite` `DoAirplaneSprite` `DoShipSprite` `DoTrainSprite` `DoCopterSprite` `DoExplosionSprite` `MakeAirCrash` `StartFire` `Destroy` `OFireZone` `MonsterHere` `CanDriveOn`——移動、碰撞、破壞全在這裡 |
| `w_tool.c` | **規則** | `check3x3` `check4x4` `check6x6` 與各 `*_tool()`：分區放置的合法性、佔地、成本、推土機、廢墟填補 |
| `w_budget.c` | 混合 | `UpdateBudget` 與撥款百分比是規則的一部分，其餘是視窗繪製 |
| `w_eval.c` | 介面 | 只是把 `s_eval.c` 算好的評分送進 Tcl |
| `w_sim.c` | 介面 | 128 個頂層定義幾乎都是 Tcl 指令繫結 |
| `w_con.c` `w_date.c` `w_editor.c` `w_graph.c` `w_inter.c` `w_keys.c` `w_map.c` `w_net.c` `w_piem.c` `w_print.c` `w_resrc.c` `w_sound.c` `w_stubs.c` `w_tk.c` `w_update.c` `w_util.c` `w_x.c` `w_cam.c` | 介面 | X11／Tcl／Tk、繪圖、音效、多人網路 |
| `g_*.c` | 介面 | 地圖與小地圖繪製 |
| `sim.c` | 混合 | `main()`、`sim_loop()`、`sim_update()`——**主迴圈的節奏在這裡**，不在 `s_sim.c` |

**這條教訓寫成規則**：檔名前綴是命名慣例，不是證據。判一個檔案是規則還是呈現，
要看它的頂層函式，不能看它叫什麼。

## 規則層入口速查

### 主迴圈與時間

| 檔案 | 函式 | 做什麼 |
|---|---|---|
| `sim.c` | `main` `sim_loop` `sim_timeout_loop` `sim_update` `sim_heat` | 程式進入點、每輪節奏、更新分派 |
| `s_sim.c` | `SimFrame` `Simulate(int mod16)` | **模擬的一格**。`Simulate` 依 `mod16` 分 16 個相位做不同的事 |
| `s_sim.c` | `DoSimInit` `SimLoadInit` `SetCommonInits` `InitSimMemory` | 初始化 |
| `s_sim.c` | `ClearCensus` `TakeCensus` `Take2Census` | 人口普查（短期／長期圖表）|
| `s_sim.c` | `CollectTax` `UpdateFundEffects` `SetValves` | 稅收、撥款效果、R/C/I 需求閥 |
| `s_sim.c` | `MapScan` | 逐格掃描的主分派，`DoRail` `DoRoad` `DoBridge` `DoFire` `DoSPZone` `DoAirport` `DoMeltdown` 都由它叫 |

### 隨機數（逐 tick 對拍的前提）

| 檔案 | 函式 | 備註 |
|---|---|---|
| `s_sim.c` | `Rand(short range)` `Rand16` `Rand16Signed` `SeedRand` `RandomlySeedRand` | 遊戲用的取數介面 |
| `rand.c` | `sim_rand` `sim_srand` | **遊戲實際使用的產生器**：12 行的 24 位元 LCG |
| `random.c` | `sim_random` `sim_srandom` `sim_initstate` `sim_setstate` | BSD `random()` 的完整移植，**遊戲沒有用到**（`grep -rn 'sim_random' *.c` 只命中它自己）|

**這是本專案能做逐 tick 對拍的關鍵**：亂數不呼叫系統 libc，就在封存裡，
而且是最簡單的那種 LCG。公式與驗證見
[`02-rng.md`](02-rng.md)（已對活的 oracle 驗證：看四個輸出就能預測其餘）。

### 每格的模擬

| 檔案 | 函式 | 做什麼 |
|---|---|---|
| `s_zone.c` | `DoZone` `DoResidential` `DoCommercial` `DoIndustrial` `DoHospChur` | 分區格的成長與衰退 |
| `s_zone.c` | `DoResIn` `DoComIn` `DoIndIn` `DoResOut` `DoComOut` `DoIndOut` | 進出人口 |
| `s_zone.c` | `EvalRes` `EvalCom` `EvalInd` `EvalLot` `GetCRVal` | 每格的吸引力評估（交通是輸入）|
| `s_zone.c` | `ResPlop` `ComPlop` `IndPlop` `ZonePlop` `BuildHouse` `MakeHosp` | 換上對應的圖塊 |
| `s_zone.c` | `RZPop` `CZPop` `IZPop` `DoFreePop` `SetZPower` | 每格人口與供電旗標 |
| `s_power.c` | `DoPowerScan` `SetPowerBit` `TestPowerBit` `TestForCond` `MoveMapSim` | 電力網傳導（堆疊式泛洪）|
| `s_traf.c` | `MakeTraf` `TryDrive` `TryGo` `FindPRoad` `FindPTele` `DriveDone` `RoadTest` | 交通生成：從一格找得到目的地就算通 |
| `s_scan.c` | `PopDenScan` `PTLScan` `CrimeScan` `FireAnalysis` | 人口密度、汙染／地價、犯罪、消防涵蓋的掃描 |
| `s_scan.c` | `DoSmooth` `DoSmooth2` `SmoothTerrain` `SmoothFSMap` `SmoothPSMap` | 各種平滑（掃描結果的擴散）|

### 評分、災難、訊息

| 檔案 | 函式 | 做什麼 |
|---|---|---|
| `s_eval.c` | `CityEvaluation` `GetScore` `DoProblems` `VoteProblems` `DoVotes` `GetAssValue` `DoPopNum` `AverageTrf` `GetUnemployment` `GetFire` | 城市評分與市民投票 |
| `s_disast.c` | `DoDisasters` `ScenarioDisaster` `MakeEarthquake` `MakeFlood` `DoFlood` `MakeFire` `SetFire` `FireBomb` `MakeMeltdown` `Vunerable` | 災難 |
| `w_sprite.c` | `MoveObjects` `DoMonsterSprite` `DoTornadoSprite` `DoAirplaneSprite` `DoShipSprite` `DoTrainSprite` `DoCopterSprite` `DoExplosionSprite` `DoBusSprite` | 精靈的每格行為 |
| `w_sprite.c` | `MakeMonster` `MonsterHere` `MakeAirCrash` `GenerateTrain` `GenerateBus` `GenerateShip` `MakeShipHere` | 精靈的產生條件 |
| `w_sprite.c` | `Destroy` `ExplodeSprite` `OFireZone` `StartFire` `CheckSpriteCollision` `CanDriveOn` `checkWet` | 破壞與碰撞 |
| `s_msg.c` | `SendMessages` `CheckGrowth` `SendMes` `SendMesAt` `DoScenarioScore` `DoLoseGame` `DoWinGame` | 訊息觸發與劇本勝敗判定 |

### 工具（玩家操作）

| 檔案 | 函式 | 做什麼 |
|---|---|---|
| `w_tool.c` | `check3x3` `check4x4` `check6x6` 與三個 `*border` | 放置合法性、佔地檢查 |
| `w_tool.c` | `bulldozer_tool` `road_tool` `rail_tool` `wire_tool` `park_tool` `residential_tool` `commercial_tool` `industrial_tool` `police_dept_tool` `fire_dept_tool` `stadium_tool` `coal_power_plant_tool` `nuclear_power_plant_tool` `seaport_tool` `airport_tool` | 每種工具 |
| `w_tool.c` | `put3x3Rubble` `put4x4Rubble` `put6x6Rubble` `tally` `checkSize` | 拆除與廢墟 |

### 地形、存檔、資料結構

| 檔案 | 函式／內容 | 做什麼 |
|---|---|---|
| `s_gen.c` | 地形產生 | 河流、湖泊、森林 |
| `s_fileio.c` | 城市存檔讀寫 | `.cty` 格式的一手證據 |
| `s_alloc.c` | `initMapArrays` | **全部地圖陣列的配置。動任何欄位前先讀這裡** |
| `headers/sim.h` | 結構、常數、圖塊編號 | 18 KB，全域宣告 |
| `headers/macros.h` | 巨集 | 4.7 KB |
| `headers/animtab.h` | 動畫表 | 16 KB |

## 逐檔 SHA-256

| 檔案 | 大小 | SHA-256 |
|---|---:|---|
| `g_ani.c` | 4154 | `f78b02a0916dd2b842b8b496aff796547889056758c9215d554bdbb5d9a019b8` |
| `g_bigmap.c` | 9441 | `b063332cce152b70c9f1f9060fe520ae1ed786f327728fcdcf774cbe9b5c35a7` |
| `g_cam.c` | 26402 | `d23196b66a2bd894d6faf110b9cf1d3e39d528557c47360877070a3b1a12f8f7` |
| `g_map.c` | 11849 | `4c146148241c15f16cc56525e8bc4714ddd80868657b55bf2121d1ec19a65273` |
| `g_setup.c` | 11197 | `d045585eebc500353ce2402cedd3bdb17404d336597a81d3e6701a9e6cdc2c4c` |
| `g_smmaps.c` | 11270 | `e2bdadb8b0da7294ec86e3120d8c8ba620c0088b63ad90222a555343cb39024a` |
| `rand.c` | 2104 | `3b1b68419f98b58429a5c89be10d48750b6aa8e23e58459159de516819cb6479` |
| `random.c` | 13317 | `7cdc4e655c39c2f4fa030f34c7661c0d8830842667d1ccb9823e4e7830b7cbb9` |
| `s_alloc.c` | 6576 | `66fa9badd297b14f59aa4d7876a58582ae16283ffb8b1fd1092ecad5eade85db` |
| `s_disast.c` | 8279 | `9cfaa9b307cce08d0e00b0da378d8230cf4b66f8968d3ba1d83b556271e270c9` |
| `s_eval.c` | 8804 | `c695f37bad5d77a11844c958da6cb3b4c8f0a2f3d3712adce1c275374a6f18c6` |
| `s_fileio.c` | 13937 | `ab7f3cfa7348477be7b7424bf601cf9e855421b7bade19c06e9eb2535786667e` |
| `s_gen.c` | 14620 | `0c592ab0d03e3a5714a157407468d9291facb5d60df575d410a02af103b6004f` |
| `sim.c` | 19090 | `f1c4df70a54590411b5e85f06593f210168b3b188c20a3125be303e18c5b6995` |
| `s_init.c` | 4895 | `a10d465bf9117f28b34285f4c099424d88c18dbf782b089ac0a383e40d6be9a2` |
| `s_msg.c` | 10513 | `10db5b366ee6367db2639c8c697d6bb2822d735c649fdecf1f25035bdc4e90c1` |
| `s_power.c` | 6777 | `b20998453327b8926028725a58d99d3eafc65791927a42688ef9d00a5b9ee43c` |
| `s_scan.c` | 13575 | `aa499244c0f6b09b759379cb48ec425f6bd3460fd92b46cf8ee1fc2c40164002` |
| `s_sim.c` | 29203 | `3700ddfec12faf61781dc93cd716bd8c913ec066775877c2b7a7a08249b07584` |
| `s_traf.c` | 8734 | `68d04a5af15543b1481b85cb1e797d1e12c9220c6c240daf4d65aace0aef90d6` |
| `s_zone.c` | 14547 | `c821022ecfd6162a96d2f229b7416f131e36c0094da6232b73b7fcc7f8436206` |
| `w_budget.c` | 9412 | `3f394d635e28a38a7c72d498fab7294266f1a24fe3bb658e977d68f8969a1c3e` |
| `w_cam.c` | 21272 | `deb398a6ff788d4080951d4e748265858f496ff133b27829fab180e50a1ec494` |
| `w_con.c` | 15193 | `e44c418b799527bfd6973d03c242f085fd1803412a08c2e370948d40134fd0e0` |
| `w_date.c` | 18192 | `583472ee383ead8bd9168bb8b9c7e7c3ae4a728904b3f091162fdb79f7eefcd6` |
| `w_editor.c` | 39189 | `cbbd3f58e4f3267feaa52571b3c7987493a48a81542a0cc539733bf0481e1ef0` |
| `w_eval.c` | 6385 | `2daade074eea7b495183587e04009b71df6975cfbdd7cd7df4b9aecd78ecff22` |
| `w_graph.c` | 22467 | `e46c7384578cf88a11af2bee5b8d923d13fb3b9fe2764fee4b618c06f50b885a` |
| `w_inter.c` | 52720 | `9215878fb9d9f743c415e4fa51b30efc242902cb9fc1bd083fc89ddf17471c76` |
| `w_keys.c` | 8900 | `3345ca893a97bf06b4843c4f1dcccc68c25bd1e86be556b186598ce7b8505ccb` |
| `w_map.c` | 14877 | `ef6699fdb29bbe38c8bcd5bb60e1573e6b42c202c4cdde81b2fdc77ba64a16bd` |
| `w_net.c` | 5053 | `5655bfbcbea5976c35c01f996c285149e3c6c2a39c5bf0a0e1b19d50718e3d2c` |
| `w_piem.c` | 66898 | `fce191f73d7d2bb6c9f9756bcf3a1d5845e56a1e56fae646553c0a2cc88f325d` |
| `w_print.c` | 4025 | `c8ec8887394d48ae0eda42b63c36ca952c38a2110c248d245ddcc57e4e723a3b` |
| `w_resrc.c` | 6102 | `6b6d2279d6afbf102b7289ed977241d1cfc8f2fa053681a8ba5c91a6a0bf8643` |
| `w_sim.c` | 30545 | `d23b0abf0170d866623049164f710d44b2e3d59618ca9946ad820f9057d7a06d` |
| `w_sound.c` | 4754 | `524ffdc65ade712c4ff90c46a7abada26ce6f3de085255766d3526b308ae07bf` |
| `w_sprite.c` | 36929 | `270b18522e5b26fd78868d384ea9f868d641a8982f8f3acd6635645e18d06a70` |
| `w_stubs.c` | 5108 | `5a429d51edb22970168b532797fe6d83ad6db785ba5bf1dcd01a4f507f16d697` |
| `w_tk.c` | 20641 | `07f172658c4784accb331253eb41f1b8a095ef0077322328b862dbec9f609f82` |
| `w_tool.c` | 36496 | `e9d591b2d5078c1114d6cbe26c1ae8f13bd82f14782a3a4ff383ada778806932` |
| `w_update.c` | 6581 | `65549b05f04aec2f8d027a82b21a2587428756083a335f306baf91a3dec1aafb` |
| `w_util.c` | 7142 | `6b2cb702bad7601a3e0696857ab13793b61c7a9c9b1df2b689391793aef7fce7` |
| `w_x.c` | 38862 | `b9d7534f28450334a0d515006ad5eaff8518c7830075ac9513909f30bde5c48e` |
| `headers/animtab.h` | 16375 | `714b62985605b2506eb65f20a23c0858a26aacf3fdf6d3364f0118431190bc37` |
| `headers/cam.h` | 6404 | `03f18a82e6caf1c01cfb0d152bb26ee3179f01c8da4d11c83b7f38535ffcfcaf` |
| `headers/mac.h` | 3789 | `9acab2de8f2335944f3b4e18706d231c017b2266c82028fcf73f9e69f200ade6` |
| `headers/macros.h` | 4732 | `09ab60891e8dce304e6f86c83ab9bc852e2bcb195cae28db8131e9870d82c2b8` |
| `headers/sim.h` | 18066 | `a164d5e780908fc03bfe29c6147347fa511572ce3c3f584f40d947f6c7048cb7` |
| `headers/view.h` | 8000 | `1817c1aeece6fc12e65081100138d6fba1c3320233d0b6f6405669fa7bf6b7df` |
