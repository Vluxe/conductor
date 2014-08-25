package conductor

type Cache struct {
	c map[string]string
	s *Server
}

func createCache(server *Server) Cache {
	return cacheHub{c: make(map[string]string), s: server}
}

func (cache *Cache) Set(key, value string) error {
	cache.c[key] = value
	// do some broadcast action.
}

func (cache *Cache) Get(key string) (string, error) {
	val := cache.c[key]
	if val != nil {
		return val
	}
	// do some broadcast action.
}

func (cache *Cache) Evict(key string) error {
	delete(cache.c, key)
	// broadcast to peers to delete key.
}
