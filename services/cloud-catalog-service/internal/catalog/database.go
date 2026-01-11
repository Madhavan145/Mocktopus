package catalog

import "moktopus/cloud-catalog-service/internal/model"

var AWSDatabaseInstances = []model.DatabaseConfig{
	// PostgreSQL
	{ID: "aws-rds-t4g-micro-pg", Provider: "aws", Service: "RDS", Engine: "postgres", InstanceClass: "db.t4g.micro", VCPUs: 2, MemoryGB: 1, IOPS: 3000, PricePerMonth: 12.10, MultiAZ: false},
	{ID: "aws-rds-t4g-small-pg", Provider: "aws", Service: "RDS", Engine: "postgres", InstanceClass: "db.t4g.small", VCPUs: 2, MemoryGB: 2, IOPS: 3000, PricePerMonth: 24.82, MultiAZ: false},
	{ID: "aws-rds-t4g-medium-pg", Provider: "aws", Service: "RDS", Engine: "postgres", InstanceClass: "db.t4g.medium", VCPUs: 2, MemoryGB: 4, IOPS: 3000, PricePerMonth: 49.64, MultiAZ: false},
	{ID: "aws-rds-r6g-large-pg", Provider: "aws", Service: "RDS", Engine: "postgres", InstanceClass: "db.r6g.large", VCPUs: 2, MemoryGB: 16, IOPS: 12000, PricePerMonth: 175.20, MultiAZ: false},
	{ID: "aws-rds-r6g-xlarge-pg", Provider: "aws", Service: "RDS", Engine: "postgres", InstanceClass: "db.r6g.xlarge", VCPUs: 4, MemoryGB: 32, IOPS: 12000, PricePerMonth: 350.40, MultiAZ: false},
	{ID: "aws-rds-r6g-2xlarge-pg", Provider: "aws", Service: "RDS", Engine: "postgres", InstanceClass: "db.r6g.2xlarge", VCPUs: 8, MemoryGB: 64, IOPS: 20000, PricePerMonth: 700.80, MultiAZ: false},
	{ID: "aws-rds-r6g-large-pg-ha", Provider: "aws", Service: "RDS", Engine: "postgres", InstanceClass: "db.r6g.large", VCPUs: 2, MemoryGB: 16, IOPS: 12000, PricePerMonth: 350.40, MultiAZ: true},
	{ID: "aws-rds-r6g-xlarge-pg-ha", Provider: "aws", Service: "RDS", Engine: "postgres", InstanceClass: "db.r6g.xlarge", VCPUs: 4, MemoryGB: 32, IOPS: 12000, PricePerMonth: 700.80, MultiAZ: true},
	// MySQL
	{ID: "aws-rds-t4g-micro-mysql", Provider: "aws", Service: "RDS", Engine: "mysql", InstanceClass: "db.t4g.micro", VCPUs: 2, MemoryGB: 1, IOPS: 3000, PricePerMonth: 12.10, MultiAZ: false},
	{ID: "aws-rds-r6g-large-mysql", Provider: "aws", Service: "RDS", Engine: "mysql", InstanceClass: "db.r6g.large", VCPUs: 2, MemoryGB: 16, IOPS: 12000, PricePerMonth: 175.20, MultiAZ: false},
	{ID: "aws-rds-r6g-xlarge-mysql", Provider: "aws", Service: "RDS", Engine: "mysql", InstanceClass: "db.r6g.xlarge", VCPUs: 4, MemoryGB: 32, IOPS: 12000, PricePerMonth: 350.40, MultiAZ: false},
	{ID: "aws-rds-r6g-large-mysql-ha", Provider: "aws", Service: "RDS", Engine: "mysql", InstanceClass: "db.r6g.large", VCPUs: 2, MemoryGB: 16, IOPS: 12000, PricePerMonth: 350.40, MultiAZ: true},
	// Aurora
	{ID: "aws-aurora-r6g-large", Provider: "aws", Service: "Aurora", Engine: "aurora-postgresql", InstanceClass: "db.r6g.large", VCPUs: 2, MemoryGB: 16, IOPS: 40000, PricePerMonth: 218.88, MultiAZ: true},
	{ID: "aws-aurora-r6g-xlarge", Provider: "aws", Service: "Aurora", Engine: "aurora-postgresql", InstanceClass: "db.r6g.xlarge", VCPUs: 4, MemoryGB: 32, IOPS: 40000, PricePerMonth: 437.76, MultiAZ: true},
	{ID: "aws-aurora-r6g-2xlarge", Provider: "aws", Service: "Aurora", Engine: "aurora-postgresql", InstanceClass: "db.r6g.2xlarge", VCPUs: 8, MemoryGB: 64, IOPS: 40000, PricePerMonth: 875.52, MultiAZ: true},
	// DynamoDB
	{ID: "aws-dynamodb-ondemand", Provider: "aws", Service: "DynamoDB", Engine: "dynamodb", InstanceClass: "On-Demand", VCPUs: 0, MemoryGB: 0, IOPS: 40000, PricePerMonth: 0, MultiAZ: true},
	{ID: "aws-dynamodb-provisioned", Provider: "aws", Service: "DynamoDB", Engine: "dynamodb", InstanceClass: "Provisioned (100 WCU)", VCPUs: 0, MemoryGB: 0, IOPS: 100, PricePerMonth: 47.45, MultiAZ: true},
}

var GCPDatabaseInstances = []model.DatabaseConfig{
	// Cloud SQL PostgreSQL
	{ID: "gcp-cloudsql-db-f1-pg", Provider: "gcp", Service: "Cloud SQL", Engine: "postgres", InstanceClass: "db-f1-micro", VCPUs: 0, MemoryGB: 0.6, IOPS: 3000, PricePerMonth: 9.37, MultiAZ: false},
	{ID: "gcp-cloudsql-db-g1-pg", Provider: "gcp", Service: "Cloud SQL", Engine: "postgres", InstanceClass: "db-g1-small", VCPUs: 0, MemoryGB: 1.7, IOPS: 3000, PricePerMonth: 27.25, MultiAZ: false},
	{ID: "gcp-cloudsql-db-n1-2-pg", Provider: "gcp", Service: "Cloud SQL", Engine: "postgres", InstanceClass: "db-custom-2-8192", VCPUs: 2, MemoryGB: 8, IOPS: 15000, PricePerMonth: 97.09, MultiAZ: false},
	{ID: "gcp-cloudsql-db-n1-4-pg", Provider: "gcp", Service: "Cloud SQL", Engine: "postgres", InstanceClass: "db-custom-4-16384", VCPUs: 4, MemoryGB: 16, IOPS: 25000, PricePerMonth: 194.18, MultiAZ: false},
	{ID: "gcp-cloudsql-db-n1-8-pg", Provider: "gcp", Service: "Cloud SQL", Engine: "postgres", InstanceClass: "db-custom-8-32768", VCPUs: 8, MemoryGB: 32, IOPS: 30000, PricePerMonth: 388.36, MultiAZ: false},
	{ID: "gcp-cloudsql-db-n1-4-pg-ha", Provider: "gcp", Service: "Cloud SQL", Engine: "postgres", InstanceClass: "db-custom-4-16384 (HA)", VCPUs: 4, MemoryGB: 16, IOPS: 25000, PricePerMonth: 388.36, MultiAZ: true},
	// Cloud SQL MySQL
	{ID: "gcp-cloudsql-db-n1-2-mysql", Provider: "gcp", Service: "Cloud SQL", Engine: "mysql", InstanceClass: "db-custom-2-8192", VCPUs: 2, MemoryGB: 8, IOPS: 15000, PricePerMonth: 97.09, MultiAZ: false},
	{ID: "gcp-cloudsql-db-n1-4-mysql", Provider: "gcp", Service: "Cloud SQL", Engine: "mysql", InstanceClass: "db-custom-4-16384", VCPUs: 4, MemoryGB: 16, IOPS: 25000, PricePerMonth: 194.18, MultiAZ: false},
	// Cloud Spanner
	{ID: "gcp-spanner-1node", Provider: "gcp", Service: "Cloud Spanner", Engine: "spanner", InstanceClass: "1 node", VCPUs: 2, MemoryGB: 0, IOPS: 10000, PricePerMonth: 657.00, MultiAZ: true},
	{ID: "gcp-spanner-3node", Provider: "gcp", Service: "Cloud Spanner", Engine: "spanner", InstanceClass: "3 nodes", VCPUs: 6, MemoryGB: 0, IOPS: 30000, PricePerMonth: 1971.00, MultiAZ: true},
	// Firestore
	{ID: "gcp-firestore", Provider: "gcp", Service: "Firestore", Engine: "firestore", InstanceClass: "Native mode", VCPUs: 0, MemoryGB: 0, IOPS: 10000, PricePerMonth: 0, MultiAZ: true},
}

var AzureDatabaseInstances = []model.DatabaseConfig{
	// Azure Database for PostgreSQL
	{ID: "azure-pg-b1ms", Provider: "azure", Service: "Azure Database for PostgreSQL", Engine: "postgres", InstanceClass: "B1ms (Burstable)", VCPUs: 1, MemoryGB: 2, IOPS: 640, PricePerMonth: 25.55, MultiAZ: false},
	{ID: "azure-pg-b2s", Provider: "azure", Service: "Azure Database for PostgreSQL", Engine: "postgres", InstanceClass: "B2s (Burstable)", VCPUs: 2, MemoryGB: 4, IOPS: 1280, PricePerMonth: 51.10, MultiAZ: false},
	{ID: "azure-pg-gp-d2s", Provider: "azure", Service: "Azure Database for PostgreSQL", Engine: "postgres", InstanceClass: "GP D2s_v3", VCPUs: 2, MemoryGB: 8, IOPS: 3200, PricePerMonth: 124.10, MultiAZ: false},
	{ID: "azure-pg-gp-d4s", Provider: "azure", Service: "Azure Database for PostgreSQL", Engine: "postgres", InstanceClass: "GP D4s_v3", VCPUs: 4, MemoryGB: 16, IOPS: 6400, PricePerMonth: 248.20, MultiAZ: false},
	{ID: "azure-pg-gp-d8s", Provider: "azure", Service: "Azure Database for PostgreSQL", Engine: "postgres", InstanceClass: "GP D8s_v3", VCPUs: 8, MemoryGB: 32, IOPS: 12800, PricePerMonth: 496.40, MultiAZ: false},
	{ID: "azure-pg-gp-d4s-ha", Provider: "azure", Service: "Azure Database for PostgreSQL", Engine: "postgres", InstanceClass: "GP D4s_v3 (HA)", VCPUs: 4, MemoryGB: 16, IOPS: 6400, PricePerMonth: 496.40, MultiAZ: true},
	// Azure SQL
	{ID: "azure-sql-s0", Provider: "azure", Service: "Azure SQL Database", Engine: "mssql", InstanceClass: "S0 (10 DTU)", VCPUs: 0, MemoryGB: 0, IOPS: 100, PricePerMonth: 15.03, MultiAZ: false},
	{ID: "azure-sql-s1", Provider: "azure", Service: "Azure SQL Database", Engine: "mssql", InstanceClass: "S1 (20 DTU)", VCPUs: 0, MemoryGB: 0, IOPS: 200, PricePerMonth: 30.05, MultiAZ: false},
	{ID: "azure-sql-s3", Provider: "azure", Service: "Azure SQL Database", Engine: "mssql", InstanceClass: "S3 (100 DTU)", VCPUs: 0, MemoryGB: 0, IOPS: 1000, PricePerMonth: 150.26, MultiAZ: false},
	{ID: "azure-sql-gp-2vc", Provider: "azure", Service: "Azure SQL Database", Engine: "mssql", InstanceClass: "GP 2 vCores", VCPUs: 2, MemoryGB: 10.2, IOPS: 5000, PricePerMonth: 370.39, MultiAZ: false},
	{ID: "azure-sql-gp-4vc", Provider: "azure", Service: "Azure SQL Database", Engine: "mssql", InstanceClass: "GP 4 vCores", VCPUs: 4, MemoryGB: 20.4, IOPS: 10000, PricePerMonth: 740.78, MultiAZ: false},
	// Cosmos DB
	{ID: "azure-cosmosdb-400ru", Provider: "azure", Service: "Cosmos DB", Engine: "cosmosdb", InstanceClass: "400 RU/s", VCPUs: 0, MemoryGB: 0, IOPS: 400, PricePerMonth: 23.36, MultiAZ: true},
	{ID: "azure-cosmosdb-1000ru", Provider: "azure", Service: "Cosmos DB", Engine: "cosmosdb", InstanceClass: "1000 RU/s", VCPUs: 0, MemoryGB: 0, IOPS: 1000, PricePerMonth: 58.40, MultiAZ: true},
}

func AllDatabaseInstances() []model.DatabaseConfig {
	all := make([]model.DatabaseConfig, 0, len(AWSDatabaseInstances)+len(GCPDatabaseInstances)+len(AzureDatabaseInstances))
	all = append(all, AWSDatabaseInstances...)
	all = append(all, GCPDatabaseInstances...)
	all = append(all, AzureDatabaseInstances...)
	return all
}

func DatabaseByProvider(provider string) []model.DatabaseConfig {
	switch provider {
	case "aws":
		return AWSDatabaseInstances
	case "gcp":
		return GCPDatabaseInstances
	case "azure":
		return AzureDatabaseInstances
	default:
		return nil
	}
}
