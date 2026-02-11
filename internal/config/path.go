package config

type PathConfig struct {
	Git   string `mapstructure:"git"`
	MySQL string `mapstructure:"mysql"`
	SSH   string `mapstructure:"ssh"`
	PSQL  string `mapstructure:"psql"`
	CURL  string `mapstructure:"curl"`
	AWS   string `mapstructure:"aws"`
	ZIP   string `mapstructure:"zip"`
	REDIS string `mapstructure:"redis"`
	TAR   string `mapstructure:"tar"`
	MSSQL string `mapstructure:"mssql"`
}
