## Needed programs

Go 1.21+

MySQL 8.0+

## Setup

1. Create the database

```sql
CREATE DATABASE wiki;
```

2. Set environment variables

```powershell
$env:MYSQL_PASSWORD = "yourpassword"
```

3. Run

```powershell
go run main.go
```

Visit [http://localhost:8080](http://localhost:8080)