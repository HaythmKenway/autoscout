package db

type Stats struct {
	Targets int
	Subs    int
	URLs    int
}

func GetStats() (Stats, error) {
	db, err := OpenDatabase()
	if err != nil {
		return Stats{}, err
	}
	defer db.Close()

	var s Stats
	err = db.QueryRow("SELECT COUNT(*) FROM targets").Scan(&s.Targets)
	if err != nil {
		return Stats{}, err
	}
	err = db.QueryRow("SELECT COUNT(*) FROM subdomain").Scan(&s.Subs)
	if err != nil {
		return Stats{}, err
	}
	err = db.QueryRow("SELECT COUNT(*) FROM urls").Scan(&s.URLs)
	if err != nil {
		return Stats{}, err
	}

	return s, nil
}
