package internal

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
)

const (
	seedForRandom    = 600
	rowDataDelimiter = "_|_"
	requestedErrFmt  = "error! touching row %d triggers this error"
)

// SillyDb responds to a query by printing generated data.
// The data is sequential hex32 numbers (primary ids) paired
// with some fake data.
type SillyDb struct {
	rnd          *rand.Rand
	numRowsInDb  int
	rowToErrorOn int
	types        []string
	names        []string
	revisions    []string
}

// NewSillyDb returns a new instance.
func NewSillyDb(numRowsInDb, rowToErrorOn int) *SillyDb {
	return &SillyDb{
		numRowsInDb:  numRowsInDb,
		rowToErrorOn: rowToErrorOn,
		rnd:          rand.New(rand.NewSource(int64(seedForRandom))),
		types:        strings.Split(strings.TrimSpace(fruits), "\n"),
		names:        strings.Split(strings.TrimSpace(asteroids), "\n"),
		revisions:    strings.Split(strings.TrimSpace(versions), "\n"),
	}
}

// DoLookupQuery prints some lines to stdout, representing a query result.
//
//goland:noinspection ALL
func (db *SillyDb) DoLookupQuery(id string) error {
	idAsInt, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	if db.rowToErrorOn == idAsInt {
		return fmt.Errorf(requestedErrFmt, db.rowToErrorOn)
	}
	if idAsInt > db.numRowsInDb {
		fmt.Fprintln(os.Stderr, "Error: #666: lookup failed")
		fmt.Fprintln(os.Stderr, "Error: Expected name")
		return nil
	}
	fmt.Print(`Findleblaster hizod f068ec82_6d28_5a3d11ac_1555eaa0 ---
  golattice vplm policy VPLM_Replication
  created 12/22/2017 2:11:12 PM
  modified 12/22/2017 2:11:45 PM
  society poet
  project ManufacturingEngineeringCS
  randomly unlocked
  locking not enforced
`)
	return nil
}

// DoScanQuery prints n lines to stdout, representing a query result.
func (db *SillyDb) DoScanQuery(offset, limit int) error {
	if limit < 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}
	for i := 1; i <= limit && db.numRowsInDb > 0; i++ {
		row := offset + i
		if db.rowToErrorOn == row {
			return fmt.Errorf(requestedErrFmt, db.rowToErrorOn)
		}
		fmt.Print(db.generateRow(row))
		db.numRowsInDb--
	}
	return nil
}

// NumRowsInDb returns the row count.
func (db *SillyDb) NumRowsInDb() int {
	return db.numRowsInDb
}

// RowToErrorOn reports the row which ill cause an error if it is accessed.
func (db *SillyDb) RowToErrorOn() int {
	return db.rowToErrorOn
}

func (db *SillyDb) generateRow(pid int) string {
	var b strings.Builder
	b.WriteString(db.types[db.rnd.Intn(len(db.types))])
	b.WriteString(rowDataDelimiter)
	b.WriteString(db.names[db.rnd.Intn(len(db.names))])
	b.WriteString(rowDataDelimiter)
	b.WriteString(db.revisions[db.rnd.Intn(len(db.revisions))])
	b.WriteString(rowDataDelimiter)
	b.WriteString(fmt.Sprintf("%032d", pid))
	b.WriteString("\n")
	return b.String()
}

const versions = `
-
---
1
2
3
4
5
6
`

// https://simple.wikipedia.org/wiki/List_of_fruits
//
//goland:noinspection ALL
const fruits = `
Abiu
Açaí
Acerola
Ackee
African cucumber
Apple
Apricot
Avocado
Banana
Bilberry
Blackberry
Blackcurrant
Black sapote
Blueberry
Boysenberry
Breadfruit
Buddha's hand
Cactus pear
Canistel
Cempedak
Cherimoya
Cherry
Chico fruit
Cloudberry
Coco De Mer
Coconut
Crab apple
Cranberry
Currant
Damson
Date
Dragonfruit
`

//goland:noinspection ALL
const asteroids = `
Ceres
Vesta
Pallas
Hygiea
Interamnia
Europa
Davida
Sylvia
Eunomia
Euphrosyne
Hektor
Juno
Camilla
Cybele
Patientia
Bamberga
Psyche
Thisbe
Doris
Fortuna
Themis
Amphitrite
Egeria
Elektra
Iris
Diotima
Hebe
Eugenia
Daphne
Metis
Herculina
Eleonora
Nemesis
Aurora
Ursula
Alauda
Hermione
Aletheia
Palma
Lachesis
`
