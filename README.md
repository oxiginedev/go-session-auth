# go-session

Implementing session authentication in golang application.

## Overview

There's a base session store interface that can be implemented by whatever you think of, `redis`, `database`, `file`, as far as your use case goes.

## Features

- [x] Session based auth
- [x] CSRF protection
- [x] Automatic redundant session cleanup (with goroutines)
- [] Encrypted session data

## Tests

Sadly, not tested (with code), still learning testing in go
