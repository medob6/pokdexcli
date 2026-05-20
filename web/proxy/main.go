package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const baseAPI = "https://pokeapi.co/api/v2"
const defaultLimit = 10

type LocationListResponse struct {
	Count    int    `json:"count"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Results  []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"results"`
}

type LocationAreaResponse struct {
	Name              string `json:"name"`
	PokemonEncounters []struct {
		Pokemon struct {
			Name string `json:"name"`
		} `json:"pokemon"`
	} `json:"pokemon_encounters"`
}

type Pokemon struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Stats []struct {
		BaseStat int `json:"base_stat"`
		Stat     struct {
			Name string `json:"name"`
		} `json:"stat"`
	} `json:"stats"`
	Sprites struct {
		FrontDefault string `json:"front_default"`
	} `json:"sprites"`
}

type Stat struct {
	Name string `json:"name"`
	Base int    `json:"base"`
}

type StoredPokemon struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Sprite string `json:"sprite"`
	Stats  []Stat `json:"stats"`
}

type Store struct {
	Caught map[string]StoredPokemon `json:"caught"`
	Player PlayerState             `json:"player"`
	path   string
}

type PlayerState struct {
	Level  int           `json:"level"`
	XP     int           `json:"xp"`
	Streak int           `json:"streak"`
	Daily  DailyProgress `json:"daily"`
}

type DailyProgress struct {
	Date          string `json:"date"`
	Explores      int    `json:"explores"`
	Catches       int    `json:"catches"`
	BonusClaimed  bool   `json:"bonus_claimed"`
	LastExploreAt string `json:"last_explore_at"`
}

type Mission struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Goal     int    `json:"goal"`
	Progress int    `json:"progress"`
	Done     bool   `json:"done"`
}

type EncounterPokemon struct {
	Name        string  `json:"name"`
	Rarity      string  `json:"rarity"`
	CatchChance float64 `json:"catch_chance"`
}

type Encounter struct {
	ID        string             `json:"id"`
	Index     int                `json:"index"`
	Area      string             `json:"area"`
	ExpiresAt time.Time          `json:"expires_at"`
	Pokemon   []EncounterPokemon `json:"pokemon"`
}

type EncounterStore struct {
	mu    sync.Mutex
	items map[string]Encounter
}

type StateResponse struct {
	Level        int            `json:"level"`
	XP           int            `json:"xp"`
	NextLevelXP  int            `json:"next_level_xp"`
	Streak       int            `json:"streak"`
	Daily        DailyProgress  `json:"daily"`
	Missions     []Mission      `json:"missions"`
	BonusClaimed bool           `json:"bonus_claimed"`
}

func NewEncounterStore() *EncounterStore {
	return &EncounterStore{items: map[string]Encounter{}}
}

func (s *EncounterStore) Save(encounter Encounter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[encounter.ID] = encounter
}

func (s *EncounterStore) Get(id string) (Encounter, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	enc, ok := s.items[id]
	if !ok {
		return Encounter{}, false
	}
	if time.Now().After(enc.ExpiresAt) {
		delete(s.items, id)
		return Encounter{}, false
	}
	return enc, true
}

func (s *EncounterStore) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for id, enc := range s.items {
		if now.After(enc.ExpiresAt) {
			delete(s.items, id)
		}
	}
}

func loadStore(path string) (*Store, error) {
	if path == "" {
		path = ".pokedex/store.json"
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	st := &Store{Caught: map[string]StoredPokemon{}, path: path}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		ensurePlayer(st)
		return st, nil
	} else if err != nil {
		return nil, err
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(st); err != nil {
		return nil, err
	}
	if st.Caught == nil {
		st.Caught = map[string]StoredPokemon{}
	}
	ensurePlayer(st)
	return st, nil
}

func (s *Store) Save() error {
	f, err := os.Create(s.path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(s)
}

func (s *Store) Add(p StoredPokemon) error {
	s.Caught[strings.ToLower(p.Name)] = p
	return s.Save()
}

func (s *Store) List() []StoredPokemon {
	out := make([]StoredPokemon, 0, len(s.Caught))
	for _, p := range s.Caught {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func ensurePlayer(store *Store) {
	if store.Player.Level <= 0 {
		store.Player.Level = 1
	}
	resetDailyIfNeeded(store)
}

func resetDailyIfNeeded(store *Store) {
	today := time.Now().Format("2006-01-02")
	if store.Player.Daily.Date != today {
		store.Player.Daily = DailyProgress{Date: today}
	}
}

func rarityFromStats(sum int) string {
	switch {
	case sum >= 500:
		return "legendary"
	case sum >= 420:
		return "rare"
	case sum >= 340:
		return "uncommon"
	default:
		return "common"
	}
}

func rarityFromName(name string) string {
	seed := 0
	for _, r := range name {
		seed += int(r)
	}
	switch {
	case seed%100 >= 95:
		return "legendary"
	case seed%100 >= 80:
		return "rare"
	case seed%100 >= 55:
		return "uncommon"
	default:
		return "common"
	}
}

func estimateCatchChance(rarity string) float64 {
	switch rarity {
	case "legendary":
		return 0.15
	case "rare":
		return 0.28
	case "uncommon":
		return 0.42
	default:
		return 0.6
	}
}

func computeCatchChance(p *Pokemon, rarity string, streak int) float64 {
	sum := 0
	for _, s := range p.Stats {
		sum += s.BaseStat
	}
	max := 600
	chance := 0.5*(1.0-float64(sum)/float64(max)) + 0.5
	if chance < 0.05 {
		chance = 0.05
	}
	switch rarity {
	case "legendary":
		chance *= 0.6
	case "rare":
		chance *= 0.8
	case "uncommon":
		chance *= 0.95
	default:
		chance *= 1.1
	}
	bonus := 0.02 * float64(streak)
	if bonus > 0.15 {
		bonus = 0.15
	}
	chance += bonus
	if chance > 0.95 {
		chance = 0.95
	}
	return chance
}

func xpForRarity(rarity string) int {
	switch rarity {
	case "legendary":
		return 60
	case "rare":
		return 35
	case "uncommon":
		return 20
	default:
		return 10
	}
}

func levelForXP(xp int) (int, int) {
	level := 1
	threshold := 100
	for xp >= threshold {
		xp -= threshold
		level++
		threshold = 100 + (level-1)*50
	}
	return level, threshold
}

func buildMissions(store *Store) []Mission {
	daily := store.Player.Daily
	return []Mission{
		{
			Key:      "scan",
			Label:    "Scan 3 locations",
			Goal:     3,
			Progress: daily.Explores,
			Done:     daily.Explores >= 3,
		},
		{
			Key:      "catch",
			Label:    "Catch 2 pokemon",
			Goal:     2,
			Progress: daily.Catches,
			Done:     daily.Catches >= 2,
		},
		{
			Key:      "streak",
			Label:    "Build a 3-catch streak",
			Goal:     3,
			Progress: store.Player.Streak,
			Done:     store.Player.Streak >= 3,
		},
	}
}

func updateLevel(store *Store) int {
	level, next := levelForXP(store.Player.XP)
	store.Player.Level = level
	return next
}

func applyDailyBonus(store *Store) int {
	missions := buildMissions(store)
	if store.Player.Daily.BonusClaimed {
		return 0
	}
	if missions[0].Done && missions[1].Done {
		store.Player.Daily.BonusClaimed = true
		store.Player.XP += 50
		return 50
	}
	return 0
}

func buildState(store *Store) StateResponse {
	next := updateLevel(store)
	return StateResponse{
		Level:        store.Player.Level,
		XP:           store.Player.XP,
		NextLevelXP:  next,
		Streak:       store.Player.Streak,
		Daily:        store.Player.Daily,
		Missions:     buildMissions(store),
		BonusClaimed: store.Player.Daily.BonusClaimed,
	}
}

func fetchLocationPage(offset, limit int) (*LocationListResponse, error) {
	u := fmt.Sprintf("%s/location-area?offset=%d&limit=%d", baseAPI, offset, limit)
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var lr LocationListResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return nil, err
	}
	return &lr, nil
}

func fetchPokemon(nameOrID string) (*Pokemon, error) {
	u := fmt.Sprintf("%s/pokemon/%s", baseAPI, nameOrID)
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch pokemon: %s", resp.Status)
	}
	var p Pokemon
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

func tryCatch(p *Pokemon, rarity string, streak int, encounterOK bool) (bool, float64) {
	chance := computeCatchChance(p, rarity, streak)
	if !encounterOK {
		chance *= 0.85
	}
	if chance < 0.05 {
		chance = 0.05
	}
	return rand.Float64() < chance, chance
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func parseIntQuery(r *http.Request, key string, def int) (int, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return def, nil
	}
	val, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s", key)
	}
	return val, nil
}

func handleLocations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	offset, err := parseIntQuery(r, "offset", 0)
	if err != nil || offset < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid offset"})
		return
	}
	limit, err := parseIntQuery(r, "limit", defaultLimit)
	if err != nil || limit <= 0 || limit > 50 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
		return
	}
	lr, err := fetchLocationPage(offset, limit)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "failed to fetch locations"})
		return
	}
	type result struct {
		Name  string `json:"name"`
		URL   string `json:"url"`
		Index int    `json:"index"`
	}
	results := make([]result, 0, len(lr.Results))
	for i, r := range lr.Results {
		results = append(results, result{Name: r.Name, URL: r.URL, Index: offset + i})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"offset":  offset,
		"limit":   limit,
		"count":   lr.Count,
		"results": results,
	})
}

func handleExplore(store *Store, encounters *EncounterStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		idx, err := parseIntQuery(r, "index", -1)
		if err != nil || idx < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid index"})
			return
		}
		limit, err := parseIntQuery(r, "limit", defaultLimit)
		if err != nil || limit <= 0 || limit > 50 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
			return
		}
		page := (idx / limit) * limit
		lr, err := fetchLocationPage(page, limit)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "failed to fetch locations"})
			return
		}
		relIndex := idx - page
		if relIndex < 0 || relIndex >= len(lr.Results) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "index out of range"})
			return
		}
		area := lr.Results[relIndex]
		resp, err := http.Get(area.URL)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "failed to fetch area"})
			return
		}
		defer resp.Body.Close()
		var areaResp LocationAreaResponse
		if err := json.NewDecoder(resp.Body).Decode(&areaResp); err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "failed to decode area"})
			return
		}
		names := make([]string, 0, len(areaResp.PokemonEncounters))
		for _, e := range areaResp.PokemonEncounters {
			names = append(names, e.Pokemon.Name)
		}
		resetDailyIfNeeded(store)
		store.Player.Daily.Explores++
		store.Player.Daily.LastExploreAt = time.Now().Format(time.RFC3339)
		_ = applyDailyBonus(store)
		_ = store.Save()

		encounters.Cleanup()
		encounterList := make([]EncounterPokemon, 0, len(names))
		for _, name := range names {
			rarity := rarityFromName(name)
			encounterList = append(encounterList, EncounterPokemon{
				Name:        name,
				Rarity:      rarity,
				CatchChance: estimateCatchChance(rarity),
			})
		}

		enc := Encounter{
			ID:        fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Intn(10000)),
			Index:     idx,
			Area:      areaResp.Name,
			ExpiresAt: time.Now().Add(60 * time.Second),
			Pokemon:   encounterList,
		}
		encounters.Save(enc)

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"index":        idx,
			"area":         areaResp.Name,
			"pokemon":      encounterList,
			"encounter_id": enc.ID,
			"expires_at":   enc.ExpiresAt.Format(time.RFC3339),
			"state":        buildState(store),
		})
	}
}

func handleCatch(store *Store, encounters *EncounterStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		var body struct {
			Name        string `json:"name"`
			EncounterID string `json:"encounter_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}
		name := strings.ToLower(strings.TrimSpace(body.Name))
		if name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
			return
		}
		resetDailyIfNeeded(store)
		encounterOK := false
		if body.EncounterID != "" {
			enc, ok := encounters.Get(body.EncounterID)
			if !ok {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "encounter expired"})
				return
			}
			for _, p := range enc.Pokemon {
				if strings.EqualFold(p.Name, name) {
					encounterOK = true
					break
				}
			}
			if !encounterOK {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "pokemon not in encounter"})
				return
			}
		}
		p, err := fetchPokemon(name)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "failed to fetch pokemon"})
			return
		}
		sum := 0
		for _, s := range p.Stats {
			sum += s.BaseStat
		}
		rarity := rarityFromStats(sum)
		caught, chance := tryCatch(p, rarity, store.Player.Streak, encounterOK)
		stored := StoredPokemon{ID: p.ID, Name: p.Name, Sprite: p.Sprites.FrontDefault}
		for _, s := range p.Stats {
			stored.Stats = append(stored.Stats, Stat{Name: s.Stat.Name, Base: s.BaseStat})
		}
		if caught {
			store.Player.Streak++
			store.Player.Daily.Catches++
			store.Player.XP += xpForRarity(rarity)
			_ = applyDailyBonus(store)
			if err := store.Add(stored); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save"})
				return
			}
		} else {
			store.Player.Streak = 0
			_ = store.Save()
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"caught":       caught,
			"pokemon":      stored,
			"rarity":       rarity,
			"catch_chance": chance,
			"state":        buildState(store),
		})
	}
}

func handlePokedex(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"pokemon": store.List()})
	}
}

func handleState(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		resetDailyIfNeeded(store)
		writeJSON(w, http.StatusOK, buildState(store))
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	store, err := loadStore("")
	if err != nil {
		log.Fatalf("failed to load store: %v", err)
	}
	encounters := NewEncounterStore()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/locations", handleLocations)
	mux.HandleFunc("/api/explore", handleExplore(store, encounters))
	mux.HandleFunc("/api/catch", handleCatch(store, encounters))
	mux.HandleFunc("/api/pokedex", handlePokedex(store))
	mux.HandleFunc("/api/state", handleState(store))

	addr := ":8080"
	log.Printf("Pokedex API listening on %s", addr)
	if err := http.ListenAndServe(addr, withCORS(mux)); err != nil {
		log.Fatal(err)
	}
}
