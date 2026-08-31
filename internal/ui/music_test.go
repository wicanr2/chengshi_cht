package ui

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/wicanr2/chengshi_cht/internal/sim"
)

func TestScanMusicTracksSortsSupportedFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"b.WAV", "a.ogg", "readme.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "hidden.ogg"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := scanMusicTracks(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a.ogg", "b.WAV"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tracks = %q, want %q", got, want)
	}
}

func TestAmbientMusicPoolsFollowCityClass(t *testing.T) {
	tracks := []string{
		"SC2000-10001.ogg", "SC2000-10005.ogg", "SC2000-10006.ogg",
		"SC2000-10009.ogg", "SC2000-10010.ogg", "SC2000-10011.ogg",
		"SC2000-10014.ogg", "SC2000-10015.ogg", "SC2000-10016.ogg",
		"SC2000-10018.ogg",
	}
	m := &musicPlayer{tracks: tracks, cur: -1}
	tests := []struct {
		class  int
		first  string
		second string
	}{
		{0, "SC2000-10010.ogg", "SC2000-10011.ogg"},
		{1, "SC2000-10005.ogg", "SC2000-10016.ogg"},
		{2, "SC2000-10006.ogg", "SC2000-10014.ogg"},
		{3, "SC2000-10009.ogg", "SC2000-10015.ogg"},
		{4, "SC2000-10001.ogg", "SC2000-10018.ogg"},
		{5, "SC2000-10001.ogg", "SC2000-10018.ogg"},
	}
	for _, tt := range tests {
		m.cur = -1
		first := m.nextAmbientTrack(tt.class)
		if first < 0 || tracks[first] != tt.first {
			t.Errorf("class %d first = %v, want %s", tt.class, first, tt.first)
			continue
		}
		m.cur = first
		second := m.nextAmbientTrack(tt.class)
		if second < 0 || tracks[second] != tt.second {
			t.Errorf("class %d second = %v, want %s", tt.class, second, tt.second)
		}
	}
}

func TestAmbientFallbackDoesNotTreatKnownDisasterTrackAsNormal(t *testing.T) {
	m := &musicPlayer{tracks: []string{
		"SC2000-10004.ogg", // 火災固定曲
		"my-own-song.WAV",
	}, cur: -1}
	if got := m.nextAmbientTrack(0); got != 1 {
		t.Fatalf("fallback = %d, want custom normal track 1", got)
	}

	// 目錄真的只有固定災難曲時仍要有聲音，不能因分類而永久靜音。
	m.tracks = []string{"SC2000-10004.ogg"}
	m.cur = -1
	if got := m.nextAmbientTrack(0); got != 0 {
		t.Fatalf("all-disaster fallback = %d, want 0", got)
	}
}

func TestDisasterCueForMessage(t *testing.T) {
	tests := []struct {
		message int
		event   int
		want    disasterMusicCue
	}{
		{-sim.MsgFire, 0, musicCueFire},
		{-sim.MsgFlood, 0, musicCueFlood},
		{-sim.MsgPlaneCrash, 0, musicCueAirCrash},
		{-sim.MsgShipwreck, 0, musicCueAirCrash},
		{-sim.MsgTrainCrash, 0, musicCueAirCrash},
		{-sim.MsgCopterCrash, 0, musicCueAirCrash},
		{-sim.MsgTornado, 0, musicCueTornado},
		{-sim.MsgEarthquake, 0, musicCueEarthquake},
		{-sim.MsgMonster, 0, musicCueMonster},
		{-sim.MsgMeltdown, 0, musicCueMeltdown},
		{sim.MsgExplosion, 3, musicCueAirRaid},
		{sim.MsgExplosion, 0, musicCueNone},
		{sim.MsgNeedRoads, 3, musicCueNone},
	}
	for _, tt := range tests {
		if got := disasterCueForMessage(tt.message, tt.event); got != tt.want {
			t.Errorf("message %d event %d cue = %v, want %v", tt.message, tt.event, got, tt.want)
		}
	}
}

func TestDisasterMusicMappingUsesApprovedFiles(t *testing.T) {
	want := map[disasterMusicCue]string{
		musicCueFire: "sc2000-10004", musicCueFlood: "sc2000-10000",
		musicCueAirCrash: "sc2000-10002", musicCueTornado: "sc2000-10003",
		musicCueEarthquake: "sc2000-10007", musicCueMonster: "sc2000-10013",
		musicCueAirRaid: "sc2000-10012", musicCueMeltdown: "sc2000-10008",
	}
	if !reflect.DeepEqual(disasterMusicStem, want) {
		t.Fatalf("disaster mapping = %#v, want %#v", disasterMusicStem, want)
	}
}

func TestDisasterInterruptsManualTrackAndDuplicateDoesNotRestart(t *testing.T) {
	tracks := []string{
		"SC2000-10010.ogg", // 村莊平時曲
		"SC2000-10004.ogg", // 火災
		"SC2000-10000.ogg", // 水災
	}
	var started []int
	m := &musicPlayer{tracks: tracks, cur: 0, on: true, mode: musicManual,
		startHook: func(i int) error {
			started = append(started, i)
			return nil
		}}
	g := &Game{world: sim.NewWorld(1), music: m}

	g.cueDisasterMusic(-sim.MsgFire)
	if m.cur != 1 || m.mode != musicDisaster || !reflect.DeepEqual(started, []int{1}) {
		t.Fatalf("fire state cur=%d mode=%v started=%v", m.cur, m.mode, started)
	}
	g.cueDisasterMusic(-sim.MsgFire)
	if !reflect.DeepEqual(started, []int{1}) {
		t.Fatalf("duplicate fire restarted track: %v", started)
	}
	g.cueDisasterMusic(-sim.MsgFlood)
	if m.cur != 2 || !reflect.DeepEqual(started, []int{1, 2}) {
		t.Fatalf("new disaster did not interrupt: cur=%d started=%v", m.cur, started)
	}
}

func TestDisasterTrackFinishReturnsToCurrentCityPool(t *testing.T) {
	tracks := []string{
		"SC2000-10004.ogg", // 火災
		"SC2000-10001.ogg", // 大都會
		"SC2000-10018.ogg", // 大都會
	}
	var started []int
	m := &musicPlayer{tracks: tracks, cur: 0, on: true, mode: musicDisaster,
		startHook: func(i int) error {
			started = append(started, i)
			return nil
		}}
	w := sim.NewWorld(2)
	w.CityClass = 5 // 巨型都會與大都會共池
	g := &Game{world: w, music: m}

	g.advanceMusicAfterFinish()
	if m.mode != musicAmbient || m.cur != 1 || !reflect.DeepEqual(started, []int{1}) {
		t.Fatalf("finish state cur=%d mode=%v started=%v", m.cur, m.mode, started)
	}
}

func TestFindMusicDirPriority(t *testing.T) {
	root := t.TempDir()
	flagDir := filepath.Join(root, "flag-music")
	saveDir := filepath.Join(root, "save")
	if err := os.MkdirAll(filepath.Join(saveDir, "music"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := FindMusicDir(flagDir, filepath.Join(saveDir, "city.cty")); got != filepath.Join(saveDir, "music") {
		t.Fatalf("fallback dir = %q", got)
	}
	if err := os.Mkdir(flagDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := FindMusicDir(flagDir, filepath.Join(saveDir, "city.cty")); got != flagDir {
		t.Fatalf("flag dir = %q", got)
	}
}
