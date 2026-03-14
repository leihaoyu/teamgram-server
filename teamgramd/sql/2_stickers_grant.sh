#!/bin/bash
# Grant the app user access to teamgram_stickers database.
# MYSQL_USER is provided by the MySQL Docker image environment.
mysql -u root -p"${MYSQL_ROOT_PASSWORD}" <<-EOSQL
    GRANT ALL PRIVILEGES ON teamgram_stickers.* TO '${MYSQL_USER}'@'%';
    FLUSH PRIVILEGES;
EOSQL
