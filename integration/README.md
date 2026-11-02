# Integration Tests

Each directory contains a scenario
* sleep: the enabling and disabling of a failpoint won't be delayed due to an ongoing sleep() action
* server: exercises the HTTP failpoint control API and checks basic functionality
