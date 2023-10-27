chown postgres:postgres $SLOW_TABLESPACE_PATH
chmod g+rwx $SLOW_TABLESPACE_PATH
create tablespace $SLOW_TABLESPACE owner postgres location $SLOW_TABLESPACE_PATH;
