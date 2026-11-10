
package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "./school.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("PRAGMA index_list(students)")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Student table indexes:")

	for rows.Next() {
		var seq, unique, partial int
		var name, origin string

		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			rows.Close()
			log.Fatal(err)
		}

		fmt.Printf("Index: %s | Unique: %d | Origin: %s\n", name, unique, origin)

		indexRows, err := db.Query("PRAGMA index_info('" + name + "')")
		if err != nil {
			rows.Close()
			log.Fatal(err)
		}

		for indexRows.Next() {
			var indexSeq, columnID int
			var columnName sql.NullString

			if err := indexRows.Scan(&indexSeq, &columnID, &columnName); err != nil {
				indexRows.Close()
				rows.Close()
				log.Fatal(err)
			}

			fmt.Printf("  Column: %s\n", columnName.String)
		}

		indexRows.Close()
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		log.Fatal(err)
	}
	rows.Close()
}
