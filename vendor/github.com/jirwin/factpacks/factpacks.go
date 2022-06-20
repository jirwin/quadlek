//go:generate protoc --go_out=. factpacks.proto
package factpacks

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"io"

	log "github.com/sirupsen/logrus"
	"github.com/golang/protobuf/proto"
)

var singularSep = regexp.MustCompile(`\s=>\s`)
var pluralSep = regexp.MustCompile(`\s->\s`)
var singularVerb = regexp.MustCompile(`\sis\s`)
var pluralVerb = regexp.MustCompile(`\sare\s`)
var forget = regexp.MustCompile(`\bforget\s`)

func (f *Fact) Output() string {
	var verb string
	if f.IsPlural {
		verb = "are"
	} else {
		verb = "is"
	}

	return fmt.Sprintf("%s %s %s", f.Name, verb, f.Value)
}

type lockingFactStore struct {
	factStore *FactStore
	factsMtx  sync.RWMutex
}

func (fs *lockingFactStore) SetFact(fact *Fact) {
	fs.factsMtx.Lock()
	defer fs.factsMtx.Unlock()

	fs.factStore.Facts[fact.Name] = fact
}

func (fs *lockingFactStore) GetFact(name string) *Fact {
	fs.factsMtx.RLock()
	defer fs.factsMtx.RUnlock()

	if val, ok := fs.factStore.Facts[name]; ok {
		return val
	}

	return nil
}

func (fs *lockingFactStore) DeleteFact(name string) {
	fs.factsMtx.Lock()
	defer fs.factsMtx.Unlock()

	delete(fs.factStore.Facts, name)
}

func (fs *lockingFactStore) LoadFactPack(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}

	buf := bufio.NewReader(f)

	done := false
	for done != true {
		line, err := buf.ReadString('\n')
		if err != nil && err == io.EOF {
			done = true
		} else if err != nil {
			break
		}

		var parts []string
		var isPlural bool
		if singularSep.MatchString(line) {
			parts = singularSep.Split(line, 2)
			isPlural = false
		} else if pluralSep.MatchString(line) {
			parts = pluralSep.Split(line, 2)
			isPlural = true
		}

		if len(parts) != 2 {
			log.Debug("Invalid fact format. Skipping.")
			continue
		}

		name := strings.TrimSpace(parts[0])
		fact := strings.TrimSpace(parts[1])

		if name == "" || fact == "" {
			log.Debug("Fact name and details can't be empty. Skipping.")
			continue
		}

		fs.SetFact(&Fact{
			Name:     name,
			Value:    fact,
			IsPlural: isPlural,
		})
	}

	return nil
}

func (fs *lockingFactStore) HumanFactSet(line string) bool {
	var parts []string
	var isPlural bool
	if singularVerb.MatchString(line) {
		parts = singularVerb.Split(line, 2)
		isPlural = false
	} else if pluralVerb.MatchString(line) {
		parts = pluralVerb.Split(line, 2)
		isPlural = true
	}

	if len(parts) != 2 {
		log.Debug("There isn't enough information to parse a line.")
		return false
	}

	fs.SetFact(&Fact{
		Name:     strings.TrimSpace(parts[0]),
		Value:    strings.TrimSpace(parts[1]),
		IsPlural: isPlural,
	})

	return true
}

func (fs *lockingFactStore) HumanFactForget(line string) bool {
	parts := forget.Split(line, 3)
	if len(parts) != 2 {
		return false
	}

	name := strings.TrimSpace(parts[1])

	if fs.GetFact(name) != nil {
		fs.DeleteFact(name)
		return true
	}

	return false
}

func (fs *lockingFactStore) HumanProcess(line string) {
	fs.HumanFactSet(line)
	fs.HumanFactForget(line)
}

func (fs *lockingFactStore) Serialize() ([]byte, error) {
	fs.factsMtx.RLock()
	defer fs.factsMtx.RUnlock()

	out, err := proto.Marshal(fs.factStore)
	if err != nil {
		log.WithField("err", err).Error("Unable to marshal factstore.")
		return nil, err
	}

	return out, nil
}

func (fs *lockingFactStore) Load(facts []byte) error {
	factStore := &FactStore{}

	if facts == nil {
		factStore.Facts = make(map[string]*Fact)
		return nil
	}

	err := proto.Unmarshal(facts, factStore)
	if err != nil {
		log.WithField("err", err).Error("Error loading facts.")
		return err
	}

	fs.factsMtx.Lock()
	defer fs.factsMtx.Unlock()
	fs.factStore = factStore

	return nil
}

func (fs *lockingFactStore) LookupFact(name string) string {
	fs.factsMtx.RLock()
	defer fs.factsMtx.RUnlock()

	fact := fs.GetFact(name)
	if fact == nil {
		return ""
	}

	return fact.Output()
}

func MakeFactStore() *lockingFactStore {
	fs := &FactStore{}
	fs.Facts = make(map[string]*Fact)
	return &lockingFactStore{
		factStore: fs,
	}
}
