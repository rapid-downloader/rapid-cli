package main

var globalStore = make(map[string]entry)

func store(id string, entry entry) {
	globalStore[id] = entry
}

func loadStored() (entry, bool) {
	for _, entry := range globalStore {
		return entry, true
	}

	return entry{}, false
}

func truncateStore() {
	for k := range globalStore {
		delete(globalStore, k)
	}
}
