package conductor

type Cache struct {
	c map[string]string
	s *Server
}

func createCache(server *Server) Cache {
	return Cache{c: make(map[string]string), s: server}
}

func (cache *Cache) Set(key, value string) error {
	cache.c[key] = value
	// do some broadcast action.
	return nil
}

func (cache *Cache) Get(key string) (string, error) {
	val := cache.c[key]
	if val != "" {
		return val, nil
	}
	// do some broadcast action.
	return val, nil
}

func (cache *Cache) Evict(key string) {
	delete(cache.c, key)
	// broadcast to peers to delete key.
}
