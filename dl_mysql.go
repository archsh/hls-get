package main

import (
	log "github.com/Sirupsen/logrus"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"fmt"
	"strings"
	"os"
)

/*
The following Table Structure is for hls-get to download from a MySQL db table.
`url` field is the source url for downloading,
`dest` field will be filled with file saved path after downloaded,
`ret_code` and `ret_msg` indicates the download result, 0 and empty message means DONE well.
*/

/*
-- ----------------------------
-- Table structure for download_list
-- ----------------------------
DROP TABLE IF EXISTS `hlsget_downloads`;
CREATE TABLE `hlsget_downloads` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `url` varchar(256) NOT NULL,
  `status` int(11) NOT NULL DEFAULT '0',
  `dest` varchar(256) DEFAULT NULL,
  `ret_code` int(11) DEFAULT '0',
  `ret_msg` varchar(128) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `url` (`url`)
) ENGINE=InnoDB AUTO_INCREMENT=393211 DEFAULT CHARSET=latin1;
*/

/****
 The following script shows load download list from epgdb_vod.pulish_movie:

 INSERT INTO hlsgetdb.hlsget_downloads (url, ret_code) SELECT `guid`, 0 FROM epgdb_vod.publish_movie WHERE `guid` <> "";
 */

type Dl_MySQL struct {
	host string
	port uint
	db string
	table string
	username string
	password string
	_pdb *sql.DB
}

func ShowMySQLSchema() {
	os.Stderr.Write([]byte("	-- ----------------------------\n"))
	os.Stderr.Write([]byte("	-- Table structure for download_list\n"))
	os.Stderr.Write([]byte("	-- ----------------------------\n"))
	os.Stderr.Write([]byte("	DROP TABLE IF EXISTS `hlsget_downloads`;\n"))
	os.Stderr.Write([]byte("	CREATE TABLE `hlsget_downloads` (\n"))
	os.Stderr.Write([]byte("	  `id` int(11) NOT NULL AUTO_INCREMENT,\n"))
	os.Stderr.Write([]byte("	  `url` varchar(256) NOT NULL,\n"))
	os.Stderr.Write([]byte("	  `status` int(11) NOT NULL DEFAULT '0',\n"))
	os.Stderr.Write([]byte("	  `dest` varchar(256) DEFAULT NULL,\n"))
	os.Stderr.Write([]byte("	  `ret_code` int(11) DEFAULT '0',\n"))
	os.Stderr.Write([]byte("	  `ret_msg` varchar(128) DEFAULT NULL,\n"))
	os.Stderr.Write([]byte("	  PRIMARY KEY (`id`),\n"))
	os.Stderr.Write([]byte("	  UNIQUE KEY `url` (`url`)\n"))
	os.Stderr.Write([]byte("	) ENGINE=InnoDB AUTO_INCREMENT=393211 DEFAULT CHARSET=latin1;\n"))
}

func NewMySQLDl(host string, port uint, db string, table string, username string, password string) *Dl_MySQL {
	dl := new(Dl_MySQL)
	dl.host = host
	dl.port = port
	dl.username = username
	dl.password = password
	dl.db = db
	dl.table = table
	var dburi string
	if password != "" {
		dburi = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", username, password, host, port, db)
	}else{
		dburi = fmt.Sprintf("%s@tcp(%s:%d)/%s", username, host, port, db)
	}

	pdb, err := sql.Open("mysql", dburi)
	if err != nil {
		return nil
	}else{
		dl._pdb = pdb
	}
	return dl
}

func (self *Dl_MySQL) NextLinks(limit int) ([]string, error) {
	sql := "SELECT id, url FROM "+self.table+" WHERE status = 0 LIMIT ?"
	ret := []string{}
	ids := []interface{}{}
	rows, err := self._pdb.Query(sql, limit)
	if nil != err {
		log.Errorln("NextLinks[1]:", err)
		return ret, err
	}
	for rows.Next() {
		var id int
		var link string
		e := rows.Scan(&id, &link)
		if nil != e {
			log.Errorln("NextLinks[2]:", e)
			break;
		}
		ret = append(ret, link)
		ids = append(ids, id)
	}
	if len(ids) > 0 {
		sql = "UPDATE "+self.table+" SET status=? WHERE id IN (?" + strings.Repeat(",?", len(ids)-1) + ")"
		args := []interface{}{1}
		//for _, i := range ids {
		//	args = append(args, i)
		//}
		args = append(args, ids...)
		_, e := self._pdb.Exec(sql, args...)
		if e!= nil {
			log.Errorln("NextLinks[3]:", e)
		}
	}
	return ret, nil
	//return nil, errors.New("Not implemented!")
}

func (self *Dl_MySQL) SubmitResult(link string, dest string, ret_code int, ret_msg string) {
	log.Infoln("DL >", link, dest, ret_code, ret_msg)
	status := 2
	if ret_code != 0 {
		status = -1
	}
	_, e := self._pdb.Exec("UPDATE "+self.table+" SET status=?, dest=?, ret_code=?, ret_msg=?  WHERE url = ?", status,
	dest, ret_code, ret_msg, link)
	if e!= nil {
		log.Errorln("SubmitResult[1]:", e)
	}
}
