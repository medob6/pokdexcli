package store
Stats []Stat `json:"stats"`
Sprite string `json:"sprite"`
}


type Stat struct {
Name string `json:"name"`
Base int `json:"base"`
}


type Store struct {
Caught map[string]Pokemon `json:"caught"`
path string
}


func New(path string) (*Store, error) {
if path == "" {
path = ".pokedex/store.json"
}
dir := filepath.Dir(path)
os.MkdirAll(dir, 0755)
st := &Store{Caught: map[string]Pokemon{}, path: path}
if _, err := os.Stat(path); err == nil {
f, err := os.Open(path)
if err == nil {
defer f.Close()
json.NewDecoder(f).Decode(st)
}
}
return st, nil
}


func (s *Store) Save() error {
f, err := os.Create(s.path)
if err != nil { return err }
defer f.Close()
enc := json.NewEncoder(f)
enc.SetIndent("", " ")
return enc.Encode(s)
}


func (s *Store) Add(p Pokemon) error {
s.Caught[p.Name] = p
return s.Save()
}


func (s *Store) Get(name string) (Pokemon, bool) {
p, ok := s.Caught[name]
return p, ok
}


func (s *Store) List() []Pokemon {
out := make([]Pokemon, 0, len(s.Caught))
for _, p := range s.Caught { out = append(out, p) }
return out
}