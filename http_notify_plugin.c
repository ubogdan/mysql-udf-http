
#include "http_notify_plugin.h"

extern int http_notify_plugin_init();
extern int http_notify_plugin_deinit();



struct st_mysql_daemon http_notify_plugin=
{ MYSQL_DAEMON_INTERFACE_VERSION  };

static struct st_mysql_sys_var *http_notify_sysvar[]= {
  MYSQL_SYSVAR(endpoint),
  MYSQL_SYSVAR(username),
  MYSQL_SYSVAR(password),
  NULL
};

mysql_declare_plugin(http_notify)
{
  MYSQL_DAEMON_PLUGIN,
  &http_notify_plugin,
  "http_notify",
  "Bogdan Ungureanu",
  "HTTP queue plugin",
  PLUGIN_LICENSE_PROPRIETARY,
  http_notify_plugin_init,    /* Plugin Init */
  http_notify_plugin_deinit,  /* Plugin Deinit */
  0x0100                      /* 1.0 */,
  NULL               ,        /* status variables                */
  http_notify_sysvar,         /* system variables                */
  NULL,                       /* config options                  */
  0,                          /* flags                           */
}
mysql_declare_plugin_end;

