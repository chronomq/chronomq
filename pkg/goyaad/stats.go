package goyaad

type Stats struct {
	// 	 "current-jobs-urgent" is the number of ready jobs with priority < 1024.

	//  - "current-jobs-ready" is the number of jobs in the ready queue.

	//  - "current-jobs-reserved" is the number of jobs reserved by all clients.

	//  - "current-jobs-delayed" is the number of delayed jobs.

	//  - "current-jobs-buried" is the number of buried jobs.

	//  - "cmd-put" is the cumulative number of put commands.

	//  - "cmd-peek" is the cumulative number of peek commands.

	//  - "cmd-peek-ready" is the cumulative number of peek-ready commands.

	//  - "cmd-peek-delayed" is the cumulative number of peek-delayed commands.

	//  - "cmd-peek-buried" is the cumulative number of peek-buried commands.

	//  - "cmd-reserve" is the cumulative number of reserve commands.

	//  - "cmd-use" is the cumulative number of use commands.

	//  - "cmd-watch" is the cumulative number of watch commands.

	//  - "cmd-ignore" is the cumulative number of ignore commands.

	//  - "cmd-delete" is the cumulative number of delete commands.

	//  - "cmd-release" is the cumulative number of release commands.

	//  - "cmd-bury" is the cumulative number of bury commands.

	//  - "cmd-kick" is the cumulative number of kick commands.

	//  - "cmd-stats" is the cumulative number of stats commands.

	//  - "cmd-stats-job" is the cumulative number of stats-job commands.

	//  - "cmd-stats-tube" is the cumulative number of stats-tube commands.

	//  - "cmd-list-tubes" is the cumulative number of list-tubes commands.

	//  - "cmd-list-tube-used" is the cumulative number of list-tube-used commands.

	//  - "cmd-list-tubes-watched" is the cumulative number of list-tubes-watched
	//    commands.

	//  - "cmd-pause-tube" is the cumulative number of pause-tube commands.

	//  - "job-timeouts" is the cumulative count of times a job has timed out.

	//  - "total-jobs" is the cumulative count of jobs created.

	//  - "max-job-size" is the maximum number of bytes in a job.

	//  - "current-tubes" is the number of currently-existing tubes.

	//  - "current-connections" is the number of currently open connections.

	//  - "current-producers" is the number of open connections that have each
	//    issued at least one put command.

	//  - "current-workers" is the number of open connections that have each issued
	//    at least one reserve command.

	//  - "current-waiting" is the number of open connections that have issued a
	//    reserve command but not yet received a response.

	//  - "total-connections" is the cumulative count of connections.

	//  - "pid" is the process id of the server.

	//  - "version" is the version string of the server.

	//  - "rusage-utime" is the cumulative user CPU time of this process in seconds
	//    and microseconds.

	//  - "rusage-stime" is the cumulative system CPU time of this process in
	//    seconds and microseconds.

	//  - "uptime" is the number of seconds since this server process started running.

	//  - "binlog-oldest-index" is the index of the oldest binlog file needed to
	//    store the current jobs.

	//  - "binlog-current-index" is the index of the current binlog file being
	//    written to. If binlog is not active this value will be 0.

	//  - "binlog-max-size" is the maximum size in bytes a binlog file is allowed
	//    to get before a new binlog file is opened.

	//  - "binlog-records-written" is the cumulative number of records written
	//    to the binlog.

	//  - "binlog-records-migrated" is the cumulative number of records written
	//    as part of compaction.

	//  - "id" is a random id string for this server process, generated when each
	//    beanstalkd process starts.

	//  - "hostname" the hostname of the machine as determined by uname.
}
