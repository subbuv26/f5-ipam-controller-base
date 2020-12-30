package sqlite

import (
	"database/sql"
	"fmt"
	"os"

	log "github.com/subbuv26/f5-ipam-controller/pkg/vlogger"
)

type DBStore struct {
	db *sql.DB
}

const (
	ALLOCATED = 0
	AVAILABLE = 1
)

func NewStore() *DBStore {
	_ = os.Remove("ipaddress-database.db")
	// SQLite is a file based database.

	log.Debug("Creating ipaddress-database.db...")
	file, err := os.Create("ipaddress-database.db") // Create SQLite file
	if err != nil {
		log.Errorf("Unable to Create DB File, %v", err)
		return nil
	}
	_ = file.Close()

	db, err := sql.Open("sqlite3", "./ipaddress-database.db")
	if err != nil {
		log.Errorf("Unable to Initialise DB, %v", err)
		return nil
	}

	err = db.Ping()
	if err != nil {
		log.Errorf("Unable to Establish Connection to DB, %v", err)
		return nil
	}

	store := &DBStore{db: db}
	if !store.CreateTables() {
		return nil
	}

	return store
}

func (store *DBStore) CreateTables() bool {
	createIPAddressTableSQL := `CREATE TABLE ipaddress_range (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"ipaddress" TEXT,
		"status" INT,
		"cidr" TEXT	
	  );`

	statement, _ := store.db.Prepare(createIPAddressTableSQL)

	_, err := statement.Exec()
	if err != nil {
		log.Errorf("Unable to Create Table 'ipaddress_range' in Database")
		return false
	}
	createARecodsTableSQL := `CREATE TABLE a_records (
		"ipaddress" TEXT PRIMARY_KEY,
		"hostname" TEXT	
	  );`

	statement, _ = store.db.Prepare(createARecodsTableSQL)

	_, err = statement.Exec()
	if err != nil {
		log.Errorf("Unable to Create  Table 'a_records' in Database")
		return false
	}
	return true
}

func (store *DBStore) InsertIP(ips []string, cidr string) {
	for _, j := range ips {
		insertIPSQL := `INSERT INTO ipaddress_range(ipaddress, status, cidr) VALUES (?, ?, ?)`

		statement, _ := store.db.Prepare(insertIPSQL)

		_, err := statement.Exec(j, AVAILABLE, cidr)
		if err != nil {
			log.Error("Unable to Insert row in Table 'ipaddress_range'")
		}
	}
}

func (store *DBStore) DisplayIPRecords() {

	row, err := store.db.Query("SELECT * FROM ipaddress_range ORDER BY id")
	if err != nil {
		log.Debugf(" ", err)
	}
	columns, err := row.Columns()
	if err != nil {
		log.Debugf(" err : ", err)
	}
	log.Debugf(" column names ", columns)
	defer row.Close()
	for row.Next() {
		var id int
		var ipaddress string
		var status int
		var cidr string
		row.Scan(&id, &ipaddress, &status, &cidr)
		log.Debugf("ipaddress_range: ", id, " ", ipaddress, " ", status, " ", cidr)
	}
}

func (store *DBStore) AllocateIP(cidr string) string {
	var ipaddress string
	var id int

	queryString := fmt.Sprintf(
		"SELECT ipaddress,id FROM ipaddress_range where status=%d AND cidr=%s order by id ASC limit 1",
		AVAILABLE,
		cidr,
	)
	err := store.db.QueryRow(queryString).Scan(&ipaddress, &id)
	if err != nil {
		log.Info("No Available IP Addresses to Allocate")
		return ""
	}

	allocateIPSql := fmt.Sprintf("UPDATE ipaddress_range set status = %d where id = ?", ALLOCATED)
	statement, _ := store.db.Prepare(allocateIPSql)

	_, err = statement.Exec(id)
	if err != nil {
		log.Errorf("Unable to update row in Table 'ipaddress_range': %v", err)
	}
	return ipaddress
}

func (store *DBStore) ReleaseIP(ip string) {
	unallocateIPSql := fmt.Sprintf("UPDATE ipaddress_range set status = %d where ipaddress = ?", AVAILABLE)
	statement, _ := store.db.Prepare(unallocateIPSql)

	_, err := statement.Exec(ip)
	if err != nil {
		log.Errorf("Unable to update row in Table 'ipaddress_range': %v", err)
	}
}

func (store *DBStore) CreateARecord(hostname, ipAddr string) bool {
	insertARecordSQL := `INSERT INTO a_records(ipaddress, hostname) VALUES (?, ?)`

	statement, _ := store.db.Prepare(insertARecordSQL)

	_, err := statement.Exec(ipAddr, hostname)
	if err != nil {
		log.Error("Unable to Insert row in Table 'a_records'")
		return false
	}
	return true
}

func (store *DBStore) DeleteARecord(hostname, ipAddr string) bool {
	deleteARecord := fmt.Sprintf("DELETE FROM a_records WHERE ipaddress = %v AND hostname = %v", ipAddr, hostname)

	statement, _ := store.db.Prepare(deleteARecord)

	_, err := statement.Exec(ipAddr, hostname)
	if err != nil {
		log.Error("Unable to Delete row from Table 'a_records'")
		return false
	}
	return true
}
