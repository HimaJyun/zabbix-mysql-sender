package main

import (
	"database/sql"
	"flag"
	"fmt"
	"github.com/adubkov/go-zabbix"
	"github.com/go-ini/ini"
	_ "github.com/go-sql-driver/mysql"
	"strings"
	"time"
)

var Status = map[string]string{
	"Bytes_received":                 "mysql.bytes[received]",
	"Bytes_sent":                     "mysql.bytes[sent]",
	"Com_select":                     "mysql.query[select]",
	"Com_insert":                     "mysql.query[insert]",
	"Com_update":                     "mysql.query[update]",
	"Com_delete":                     "mysql.query[delete]",
	"Com_replace":                    "mysql.query[replace]",
	"Com_begin":                      "mysql.query[begin]",
	"Com_commit":                     "mysql.query[commit]",
	"Com_rollback":                   "mysql.query[rollback]",
	"Select_full_join":               "mysql.select[full_join]",
	"Select_full_range_join":         "mysql.select[full_range_join]",
	"Select_range":                   "mysql.select[range]",
	"Select_range_check":             "mysql.select[range_check]",
	"Select_scan":                    "mysql.select[scan]",
	"Innodb_buffer_pool_pages_data":  "mysql.innodb.buffer[data]",
	"Innodb_buffer_pool_pages_free":  "mysql.innodb.buffer[free]",
	"Innodb_buffer_pool_pages_dirty": "mysql.innodb.buffer[dirty]",
	"Innodb_buffer_pool_pages_misc":  "mysql.innodb.buffer[misc]",
	"Innodb_data_reads":              "mysql.innodb.data[read]",
	"Innodb_data_writes":             "mysql.innodb.data[write]",
	"Innodb_data_fsyncs":             "mysql.innodb.data[fsync]",
	"Innodb_log_writes":              "mysql.innodb.log[write]",
	"Innodb_os_log_fsyncs":           "mysql.innodb.log[fsync]",
	"Threads_cached":                 "mysql.thread[cached]",
	"Threads_connected":              "mysql.thread[connected]",
	"Threads_created":                "mysql.thread[created]",
	"Threads_running":                "mysql.thread[running]",
	"Questions":                      "mysql.query[questions]",
	"Slow_queries":                   "mysql.query[slow_queries]",
	"Memory_used":                    "mysql.memory",
	"Threadpool_idle_threads":        "mysql.thread[pool_idle]",
	"Threadpool_threads":             "mysql.thread[pool_threads]",
}

func main() {
	var (
		mysqlHost  = flag.String("my-host", "127.0.0.1", "mysql host")
		mysqlPort  = flag.Int("my-port", 3306, "mysql port")
		mysqlUser  = flag.String("my-user", "root", "mysql user")
		mysqlPass  = flag.String("my-pass", "", "mysql password")
		file       = flag.String("defaults-extra-file", "", "mysql config")
		zabbixAddr = flag.String("z", "127.0.0.1", "zabbix server")
		zabbixPort = flag.Int("p", 10051, "zabbix port")
		zabbixHost = flag.String("s", "localhost", "zabbix host name")
		debug      = flag.Bool("debug", false, "debug")
	)
	flag.Parse()

	if *file != "" {
		cfg, err := ini.Load(*file)
		if err != nil {
			fmt.Printf("Filed to read file: %s", *file)
			panic(err)
		}
		sec := cfg.Section("client")

		*mysqlHost = sec.Key("host").MustString(*mysqlHost)
		*mysqlPort = sec.Key("port").MustInt(*mysqlPort)
		*mysqlUser = sec.Key("user").MustString(*mysqlUser)
		*mysqlPass = sec.Key("password").MustString(*mysqlPass)
	}

	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/information_schema", *mysqlUser, *mysqlPass, *mysqlHost, *mysqlPort))
	if err != nil {
		println("Database connection error")
		panic(err.Error())
	}
	defer db.Close()

	data := getStatus(db)
	if *debug {
		for k, v := range *data {
			println(k + ": " + v)
		}
		return
	}
	send(*zabbixAddr, *zabbixPort, *zabbixHost, data)
	println(0)
}

func getStatus(db *sql.DB) *map[string]string {
	data := map[string]string{}

	var builder strings.Builder
	builder.Grow(1024)
	builder.WriteString("SHOW GLOBAL STATUS WHERE `Variable_name` IN (")

	first := true
	for e := range Status {
		if first {
			first = false
		} else {
			builder.WriteByte(',')
		}

		builder.WriteByte('\'')
		builder.WriteString(e)
		builder.WriteByte('\'')
	}

	builder.WriteByte(')')

	rows, err := db.Query(builder.String())
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var value string
		err := rows.Scan(&name, &value)
		if err != nil {
			panic(err.Error())
		}

		data[Status[name]] = value
	}

	return &data
}

func send(server string, port int, host string, data *map[string]string) {
	unixtime := time.Now().Unix()
	var metrics []*zabbix.Metric

	for k, v := range *data {
		metrics = append(metrics, zabbix.NewMetric(host, k, v, unixtime))
	}
	packet := zabbix.NewPacket(metrics)

	_, err := zabbix.NewSender(server, port).Send(packet)
	if err != nil {
		panic(err.Error())
	}
}
