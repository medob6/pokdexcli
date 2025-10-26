package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const baseAPI = "https://pokeapi.co/api/v2"

type LocationListResponse struct {
	Count    int `json:"count"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Results  []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"results"`
}

type Pokemon struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Stats []struct {
		BaseStat int `json:"base_stat" json:"base_stat"`
		Stat     struct {
			Name string `json:"name"`
		} `json:"stat"`
	} `json:"stats"`
	// Add sprites if desired
	Sprites struct {
		FrontDefault string `json:"front_default"`
	} `json:"sprites"`
}

type PokedexStore struct {
	Caught map[string]Pokemon `json:"caught"`
}

func ensureStoreDir() (string, error) {
	dir := filepath.Join(".", ".pokedex")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

func loadStore() (*PokedexStore, error) {
	dir, err := ensureStoreDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "store.json")
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return &PokedexStore{Caught: map[string]Pokemon{}}, nil
	} else if err != nil {
		return nil, err
	}
	defer f.Close()
	var s PokedexStore
	if err := json.NewDecoder(f).Decode(&s); err != nil && err != io.EOF {
		return nil, err
	}
	if s.Caught == nil {
		s.Caught = map[string]Pokemon{}
	}
	return &s, nil
}

func saveStore(s *PokedexStore) error {
	dir, err := ensureStoreDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "store.json")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(s)
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

// simple wrapper to get pokemon by name or id
func fetchPokemon(nameOrID string) (*Pokemon, error) {
	u := fmt.Sprintf("%s/pokemon/%s", baseAPI, nameOrID)
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch pokemon: %s", resp.Status)
	}
	var p Pokemon
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

func tryCatch(p *Pokemon) bool {
	// basic formula: chance depends on sum of base stats (lower stats easier)
	sum := 0
	for _, s := range p.Stats {
		sum += s.BaseStat
	}
	// normalize: higher sum => lower chance
	max := 600 // rough upper bound
	chance := 0.5 * (1.0 - float64(sum)/float64(max)) // between -0.5 and 0.5
	chance = chance + 0.5                                // bias so it's between 0 and 1
	if chance < 0.05 {
		chance = 0.05
	}
	r := rand.Float64()
	return r < chance
}

func printHelp() {
	fmt.Println(`Commands:
  map         - show next page of location areas (10 per page)
  mapb        - previous page of location areas
  explore N   - explore location index N (the index from map)
  catch name  - attempt to catch pokemon by name or id (eg: catch pikachu)
  inspect name - show stats for captured pokemon
  pokedex     - list caught pokemon
  help        - show this message
  exit        - quit
`)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	store, err := loadStore()
	if err != nil {
		fmt.Println("failed to load store:", err)
		return
	}
	defer saveStore(store)

	reader := bufio.NewScanner(os.Stdin)
	fmt.Println("Welcome to Pokedex CLI! Type 'help' to see commands.")
	pageOffset := 0
	pageLimit := 10
	var lastLocations []string

	for {
		fmt.Print("> ")
		if !reader.Scan() {
			break
		}
		line := strings.TrimSpace(reader.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		cmd := strings.ToLower(parts[0])

		switch cmd {
		case "help":
			printHelp()
		case "exit":
			fmt.Println("Goodbye!")
			return
		case "map":
			lr, err := fetchLocationPage(pageOffset, pageLimit)
			if err != nil {
				fmt.Println("error fetching map:", err)
				continue
			}
			lastLocations = nil
			for i, r := range lr.Results {
				idx := pageOffset + i
				fmt.Printf("[%d] %s\n", idx, r.Name)
				lastLocations = append(lastLocations, r.Name)
			}
			pageOffset += pageLimit
		case "mapb":
			if pageOffset <= pageLimit {
				fmt.Println("No previous page.")
				continue
			}
			pageOffset -= pageLimit * 2
			if pageOffset < 0 {
				pageOffset = 0
			}
			lr, err := fetchLocationPage(pageOffset, pageLimit)
			if err != nil {
				fmt.Println("error fetching map:", err)
				continue
			}
			lastLocations = nil
			for i, r := range lr.Results {
				idx := pageOffset + i
				fmt.Printf("[%d] %s\n", idx, r.Name)
				lastLocations = append(lastLocations, r.Name)
			}
			pageOffset += pageLimit
		case "explore":
			if len(parts) < 2 {
				fmt.Println("usage: explore <index>")
				continue
			}
			idxStr := parts[1]
			idx, err := strconv.Atoi(idxStr)
			if err != nil {
				fmt.Println("index must be a number")
				continue
			}
			// fetch the page that contains idx
			pg := (idx / pageLimit) * pageLimit
			lr, err := fetchLocationPage(pg, pageLimit)
			if err != nil {
				fmt.Println("error:", err)
				continue
			}
			relIndex := idx - pg
			if relIndex < 0 || relIndex >= len(lr.Results) {
				fmt.Println("index out of range")
				continue
			}
			area := lr.Results[relIndex]
			fmt.Printf("Exploring %s ...\n", area.Name)
			// simplified: call location-area endpoint to get pokemon encounters
			u := area.URL
			resp, err := http.Get(u)
			if err != nil {
				fmt.Println("failed to get area:", err)
				continue
			}
			var j map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&j); err != nil {
				fmt.Println("error decoding:", err)
				resp.Body.Close()
				continue
			}
			resp.Body.Close()
			// parse pokemon_encounters
			if encounters, ok := j["pokemon_encounters"].([]interface{}); ok {
				for i, e := range encounters {
					if m, ok := e.(map[string]interface{}); ok {
						if pObj, ok := m["pokemon"].(map[string]interface{}); ok {
							name := pObj["name"]
							fmt.Printf("  %d: %v\n", i+1, name)
						}
					}
				}
			} else {
				fmt.Println("No pokemon found here.")
			}
		case "catch":
			if len(parts) < 2 {
				fmt.Println("usage: catch <name or id>")
				continue
			}
			name := strings.ToLower(parts[1])
			p, err := fetchPokemon(name)
			if err != nil {
				fmt.Println("couldn't fetch pokemon:", err)
				continue
			}
			ok := tryCatch(p)
			if ok {
				store.Caught[p.Name] = *p
				if err := saveStore(store); err != nil {
					fmt.Println("warning: save failed:", err)
				}
				fmt.Printf("Caught %s! (id=%d)\n", p.Name, p.ID)
			} else {
				fmt.Printf("Failed to catch %s.\n", p.Name)
			}
		case "inspect":
			if len(parts) < 2 {
				fmt.Println("usage: inspect <name>")
				continue
			}
			name := strings.ToLower(parts[1])
			if p, ok := store.Caught[name]; ok {
				fmt.Printf("Name: %s  ID: %d\n", p.Name, p.ID)
				fmt.Println("Stats:")
				for _, s := range p.Stats {
					fmt.Printf("  %s: %d\n", s.Stat.Name, s.BaseStat)
				}
				if p.Sprites.FrontDefault != "" {
					fmt.Println("Sprite:", p.Sprites.FrontDefault)
				}
			} else {
				fmt.Println("You haven't caught that pokemon yet.")
			}
		case "pokedex":
			if len(store.Caught) == 0 {
				fmt.Println("No pokemon caught yet.")
				continue
			}
			for _, p := range store.Caught {
				fmt.Printf("- %s (id=%d)\n", p.Name, p.ID)
			}
		default:
			fmt.Println("unknown command:", cmd)
			printHelp()
		}
	}
}
