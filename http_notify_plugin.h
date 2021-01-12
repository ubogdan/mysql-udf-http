#include <stdio.h>
#include <mysql.h>
#include <mysql/plugin.h>

// Config Variables
char *endpoint_gvar;
char *username_gvar;
char *password_gvar;

static MYSQL_SYSVAR_STR(endpoint, endpoint_gvar, PLUGIN_VAR_RQCMDARG | PLUGIN_VAR_READONLY | PLUGIN_VAR_MEMALLOC,  "Endpoint url https://endpoint.domain.com/webhook", NULL, NULL, "");
static MYSQL_SYSVAR_STR(username, username_gvar, PLUGIN_VAR_OPCMDARG | PLUGIN_VAR_READONLY | PLUGIN_VAR_MEMALLOC,  "Auth username", NULL, NULL, "");
static MYSQL_SYSVAR_STR(password, password_gvar, PLUGIN_VAR_OPCMDARG | PLUGIN_VAR_READONLY | PLUGIN_VAR_MEMALLOC,  "Auth password", NULL, NULL, "");



// Status Variables
//long long *status_var = 0;

