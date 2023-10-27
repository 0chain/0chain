chown postgres:postgres $SLOW_TABLE_SPACE_PATH
chmod g+rwx $SLOW_TABLE_SPACE_PATH
create tablespace $SLOW_TABLE_SPACE owner postgres location $SLOW_TABLE_SPACE_PATH;
