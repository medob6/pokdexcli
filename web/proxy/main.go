package main
lastPage = append(lastPage, r.Name)

pageOff += pageLimit


case "mapb":
if pageOff <= pageLimit { fmt.Println("No previous page."); continue }
pageOff -= pageLimit * 2
if pageOff < 0 { pageOff = 0 }
b, err := client.Get("/location-area?offset="+strconv.Itoa(pageOff)+"&limit="+strconv.Itoa(pageLimit), nil)
if err != nil { fmt.Println("err:", err); continue }
var lr struct { Results []struct{ Name string `json:"name"`; URL string `json:"url"` } `json:"results"` }
json.Unmarshal(b, &lr)
lastPage = nil
for i, r := range lr.Results { fmt.Printf("[%d] %s\n", pageOff+i, r.Name); lastPage = append(lastPage, r.Name) }
pageOff += pageLimit


case "explore":
if len(parts) < 2 { fmt.Println("usage: explore <index>); continue" ) }
idx, err := strconv.Atoi(parts[1])
if err != nil { fmt.Println("index number required"); continue }
pg := (idx / pageLimit) * pageLimit
b, err := client.Get("/location-area?offset="+strconv.Itoa(pg)+"&limit="+strconv.Itoa(pageLimit), nil)
if err != nil { fmt.Println("err:", err); continue }
var lr struct { Results []struct{ Name string `json:"name"`; URL string `json:"url"` } `json:"results"` }
json.Unmarshal(b, &lr)
rel := idx - pg
if rel < 0 || rel >= len(lr.Results) { fmt.Println("index out of range"); continue }
area := lr.Results[rel]
fmt.Println("Exploring", area.Name)
resp, err := http.Get(area.URL)
if err != nil { fmt.Println("err:", err); continue }
var j map[string]interface{}
json.NewDecoder(resp.Body).Decode(&j)
resp.Body.Close()
if enc, ok := j["pokemon_encounters"].([]interface{}); ok {
for i, e := range enc {
m := e.(map[string]interface{})
p := m["pokemon"].(map[string]interface{})
fmt.Printf(" %d: %s\n", i+1, p["name"].(string))
}
} else { fmt.Println("No pokemon here.") }


case "catch":
if len(parts) < 2 { fmt.Println("usage: catch <name>"); continue }
name := strings.ToLower(parts[1])
p, err := fetchPokemon(client, name)
if err != nil { fmt.Println("err:", err); continue }
if tryCatch(p) {
sp := store.Pokemon{ID: p.ID, Name: p.Name, Sprite: p.Sprites.FrontDefault}
for _, s := range p.Stats { sp.Stats = append(sp.Stats, store.Stat{Name: s.Stat.Name, Base: s.BaseStat}) }
st.Add(sp)
fmt.Println("Caught", p.Name)
} else { fmt.Println("Failed to catch", p.Name) }


case "inspect":
if len(parts) < 2 { fmt.Println("usage: inspect <name>"); continue }
name := strings.ToLower(parts[1])
if pk, ok := st.Get(name); ok {
fmt.Printf("%s (id=%d)\n", pk.Name, pk.ID)
for _, s := range pk.Stats { fmt.Printf(" %s: %d\n", s.Name, s.Base) }
if pk.Sprite != "" { fmt.Println("Sprite:", pk.Sprite) }
} else { fmt.Printl("")}
