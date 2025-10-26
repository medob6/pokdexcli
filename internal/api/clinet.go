package api


// Simple disk cache: store raw JSON responses under .pokedex/cache/<sha>.json
type Client struct {
CacheDir string
TTL time.Duration
HTTP *http.Client
}


func NewClient() *Client {
dir := ".pokedex/cache"
os.MkdirAll(dir, 0755)
return &Client{CacheDir: dir, TTL: 12 * time.Hour, HTTP: http.DefaultClient}
}


func (c *Client) cachePath(key string) string {
h := sha1.Sum([]byte(key))
return filepath.Join(c.CacheDir, hex.EncodeToString(h[:]) + ".json")
}


func (c *Client) getCached(key string) ([]byte, bool) {
p := c.cachePath(key)
st, err := os.Stat(p)
if err == nil {
if time.Since(st.ModTime()) < c.TTL {
b, err := os.ReadFile(p)
if err == nil {
return b, true
}
}
}
return nil, false
}


func (c *Client) setCache(key string, b []byte) error {
p := c.cachePath(key)
return os.WriteFile(p, b, 0644)
}


func (c *Client) Get(path string, query url.Values) ([]byte, error) {
u := BaseAPI + path
if len(query) > 0 {
u = u + "?" + query.Encode()
}
if b, ok := c.getCached(u); ok {
return b, nil
}
resp, err := c.HTTP.Get(u)
if err != nil {
return nil, err
}
defer resp.Body.Close()
if resp.StatusCode != 200 {
return nil, fmt.Errorf("bad status: %s", resp.Status)
}
b, err := io.ReadAll(resp.Body)
if err != nil {
return nil, err
}
_ = c.setCache(u, b)
return b, nil
}


// helper: fetch and decode JSON
func (c *Client) FetchJSON(path string, query url.Values, out interface{}) error {
b, err := c.Get(path, query)
if err != nil {
return err
}
return json.Unmarshal(b, out)
}