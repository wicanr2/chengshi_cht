# SimCity 2000 DOS 音樂的本機匯入紀錄

**日期：2026-08-31。用途：本機完整版的可選 polish，不是公開發行資產。**

## 決策與真實性邊界

- 使用者指定從其自備的 `SimCity_2000_Special_Edition_DOS.zip` 擷取音樂並轉成 OGG。
- 這些曲目屬於《模擬城市 2000》，不是 1989 DOS《模擬城市》的原版音樂；遊戲與文件
  必須繼續明示這是跨作品加入的背景音樂。
- 原始 XMI、轉換後 MIDI／WAV／OGG 全部留在 gitignore 的 `workplace/` 或 `music/`，
  不進 Git、公開 `release/` 或未取得授權的公開位置。

## 輸入與格式證據

| 項目 | 值 |
|---|---|
| ZIP SHA-256 | `e1901faa8a0941acc9a3b92ed1ea1ef4135a25312a24259639e388509b2df359` |
| `DOS/SC2000.DAT` SHA-256 | `59c73d172515dd4d1eabc4dd9a3d3bc0c0a69435f40c529c07c771a51b55f86e` |
| `SC2000.DAT` 大小 | 2,629,981 bytes |
| 資源目錄 | 399 筆；每筆 12-byte DOS 名稱加 little-endian `u32` offset |
| XMI 資源 | 75 筆，四組前綴 `1／2／3／5` |

**已確認：** `tools/sc2kextract` 由第一筆 offset 反推出目錄長度，逐筆檢查名稱、
offset 單調性與檔案界限後才擷取，不用裸特徵掃描切檔。

**強證據：** 保存社群對 `SC2000.CFG` 的 `MUSICFILES` 對照指出 `1` 是 General MIDI、
`2` 是 Sound Blaster／FM、`3` 是 MT-32、`5` 是 Wave Blaster。本次採 `10000–10018.XMI`；
`10017` 被歸為音效用途，而且實際渲染為 258.7 秒、平均約 -45.5 dB，故不列入背景播放。
參考：<https://www.vgmpf.com/Wiki/index.php?title=SimCity_2000_(DOS)>。

## 轉換鏈

```text
玩家自備 ZIP
  → SC2000.DAT
  → tools/sc2kextract（75 個 XMI）
  → General MIDI 組 10000–10018（排除 10017）
  → XMI2MID 4d3ba6abc130fbed01e8b62d3b0e163857ee2946
  → MIDI
  → FluidSynth 2.3.1 + FluidR3_GM.sf2，48 kHz、gain 0.1
  → FFmpeg 5.1.9，loudnorm I=-20／TP=-2／LRA=11
  → Ogg Vorbis quality 5、48 kHz、stereo
```

- XMI2MID 授權：MIT；來源 <https://github.com/mattseabrook/XMI2MID>。
- SoundFont：Debian `fluid-soundfont-gm 3.1-5.3`；`FluidR3_GM.sf2` SHA-256
  `74594e8f4250680adf590507a306655a299935343583256f3b722c48a1bc1cb0`。
- 工具 image：`simcity-sc2k-audio:bookworm-r1`，建置來源
  `docker/sc2k-audio.Dockerfile`。
- 18 個 OGG 的逐檔 SHA-256 保存在本機
  `workplace/sc2000-music/manifest.sha256`。

## 技術驗收與限制

- 18／18 都可由 FFmpeg 解碼，codec 為 Vorbis、48 kHz、stereo。
- 時長 17.897–194.749 秒；整合響度約 -20.9 至 -17.4 LUFS；最高量得峰值 -1.4 dBFS，
  沒有數位滿刻度峰值。
- 檔案與目錄均為目前使用者 `1000:1000`，沒有 root-owned 輸出。
- 尚需使用者實際聆聽，確認音色、曲序、短曲是否保留及長時間疲勞；技術解碼不能取代
  人耳驗收。
