/*
Copyright 2023 Milan Suk

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this db except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/


#define SkyAltInt64 long long unsigned int

char SkyAlt_initGlobal(void)
{
#ifdef WIN32
	struct WSAData wsaData;
	if (WSAStartup(MAKEWORD(1, 1), &wsaData) == SOCKET_ERROR)
		return 0;
#endif
	return 1;
}

void SkyAlt_freeGlobal(void)
{
#ifdef WIN32
	WSACleanup();
#endif
}


typedef struct SkyAltAttr_s {
	char *Name;
	char *Value;

	char *Gui_type;
	char *Gui_options;
	char Gui_ReadOnly; 	//bool
	char *Error;
} SkyAltAttr;


typedef struct SkyAltClient_s
{
	SkyAltAttr** Attrs;
	SkyAltInt64 num_Attrs;

	int socket;

} SkyAltClient;

SkyAltClient* NewSkyAltClient() {
	SkyAltClient* client = calloc(1, sizeof(SkyAltClient));
	return client;
}

void _SkyAltClient_socketFree(SkyAltClient *self)
{
	if (self->socket > 0)
	{
#ifdef WIN32
		closesocket(self->socket);
#else
		close(self->socket);
#endif
	}
	self->socket = -1;
}

void DeleteSkyAltClient(SkyAltClient* client) {

	//close connection
	_SkyAltClient_socketFree(client);

	//free data
	for(int i=0; i < client->num_Attrs; i++) {
		SkyAltAttr* attr = client->Attrs[i];
		free(attr->Name);
		free(attr->Value);
		free(attr->Gui_type);
		free(attr->Gui_options);
		free(attr->Error);

		memset(attr, 0, sizeof(SkyAltAttr));
		free(attr);
	}
	free(client->Attrs);

	memset(client, 0, sizeof(SkyAltClient));
	free(client);
}

SkyAltAttr* SkyAltClient_FindAttr(SkyAltClient* client, const char* name) {
	for(int i=0; i < client->num_Attrs; i++) {
		SkyAltAttr* attr = client->Attrs[i];
		if(!strcmp(attr->Name, name)) {
			return attr;
		}
	}
	return 0;
}

SkyAltAttr* SkyAltClient_AddAttr(SkyAltClient* client, const char* name, const char* value) {
	SkyAltAttr *attr = calloc(1, sizeof(SkyAltAttr));

	attr->Name = calloc(1, strlen(name)+1);
	attr->Value = calloc(1, strlen(value)+1);
	strcpy(attr->Name, name);
	strcpy(attr->Value, value);

	client->num_Attrs++;
	client->Attrs = realloc(client->Attrs, client->num_Attrs*sizeof(SkyAltAttr*));
	client->Attrs[client->num_Attrs-1] = attr;

	return attr;
}

SkyAltAttr* SkyAltClient_AddAttrOut(SkyAltClient* client, const char* name, const char* value) {
	SkyAltAttr* attr = SkyAltClient_AddAttr(client, name, value);
	attr->Gui_ReadOnly = 1;
	return attr;
}

char _SkyAltClient_send(SkyAltClient *client, const void *data, const int size) {
	int b = 0;
	while (b < size)
	{
		const int nbyte = send(client->socket, ((char *)data) + b, size - b, 0);
		if (nbyte < 0)
		{
			printf("send() failed\n");
			break;
		}
		b += nbyte;
	}

	return b == size;
}

char _SkyAltClient_recv(SkyAltClient *client, void *data, const int bytes) {
	int b = 0;
	while (b < bytes)
	{
		const int nbyte = recv(client->socket, ((char *)data) + b, bytes - b, 0);

		if (nbyte == 0)
		{
			_SkyAltClient_socketFree(client); // close
			return 0;
		}

		if (nbyte < 0)
		{
			printf("recv() failed\n");
			break;
		}

		b += nbyte;
	}
	return b == bytes;
}


char _SkyAltClient_recvPair(SkyAltClient* client, char **name, char **value) {

	//name
	SkyAltInt64 name_bytes;
	if(!_SkyAltClient_recv(client, &name_bytes, sizeof(name_bytes))) {	//size
		return 0;
	}
	*name = calloc(1, name_bytes+1);
	if(!_SkyAltClient_recv(client, *name, name_bytes)) {	//data
		return 0;
	}

	//value
	SkyAltInt64 value_bytes;
	if(!_SkyAltClient_recv(client, &value_bytes, sizeof(value_bytes))) {	//size
		free(*name);
		*name = 0;
		return 0;
	}
	*value = calloc(1, value_bytes+1);
	if(!_SkyAltClient_recv(client, *value, value_bytes)) {	//data
		free(*name);
		*name = 0;
		free(*value);
		*value = 0;
		return 0;
	}
	return 1;
}
char _SkyAltClient_recvPairNumber(SkyAltClient* client, char **name, int *value) {

	char *val = 0;
	if(!_SkyAltClient_recvPair(client, name, &val)) {
		return 0;
	}
	*value = atoi(val);
	free(val);
	return 1;
}



char _SkyAltClient_sendPair(SkyAltClient* client, const char *name, const char *value) {

	//name
	SkyAltInt64 name_bytes = strlen(name);
	if(!_SkyAltClient_send(client, &name_bytes, sizeof(name_bytes))) {	//size
		return 0;
	}
	if(!_SkyAltClient_send(client, name, name_bytes)) {	//data
		return 0;
	}

	//value
	SkyAltInt64 value_bytes = strlen(name);
	if(!_SkyAltClient_send(client, &value_bytes, sizeof(value_bytes))) {	//size
		return 0;
	}
	if(!_SkyAltClient_send(client, value, value_bytes)) {	//data
		return 0;
	}

	return 1;
}

char _SkyAltClient_sendPairNumber(SkyAltClient* client, const char *name, SkyAltInt64 value) {

	char t[128];
	snprintf(t, 127, "%lld", value);
	return _SkyAltClient_sendPair(client, name, t);
}



char _SkyAltClient_sendProgress(SkyAltClient* client, double proc, const char *description, const char *err) {

	if(!_SkyAltClient_sendPairNumber(client, "progress", 3)){
		return 0;
	}

	char pr[64];
	snprintf(pr, 64, "%f", proc);
	if(!_SkyAltClient_sendPair(client, "proc", pr)){
		return 0;
	}
	if(!_SkyAltClient_sendPair(client, "desc", description)){
		return 0;
	}
	if(!_SkyAltClient_sendPair(client, "error", err)){
		return 0;
	}

	return 1;
}


char _SkyAltClient_sendAttrs(SkyAltClient* client) {

	if(!_SkyAltClient_sendPairNumber(client, "attrs", client->num_Attrs)){
		return 0;
	}

	for(int i=0; i < client->num_Attrs; i++) {
		SkyAltAttr* attr = client->Attrs[i];

		//num pairs
		if(!_SkyAltClient_sendPairNumber(client, attr->Name, 5)){
			return 0;
		}

		//values
		if(!_SkyAltClient_sendPair(client, "value", attr->Value)){
			return 0;
		}
		if(!_SkyAltClient_sendPair(client, "gui_type", attr->Gui_type)){
			return 0;
		}
		if(!_SkyAltClient_sendPair(client, "gui_options", attr->Gui_options)){
			return 0;
		}
		if(!_SkyAltClient_sendPair(client, "error", attr->Error)){
			return 0;
		}
		if(!_SkyAltClient_sendPairNumber(client, "gui_read_only", attr->Gui_ReadOnly)){
			return 0;
		}
	}

	return 1;
}


char SkyAltClient_Start(SkyAltClient* client, const char* uid, const char* portStr) {

	const int port = atoi(portStr);
	if(port == 0) {
		printf("port==0\n");
		return 0;
	}

	struct sockaddr_in6 addr;
	memset(&addr, '\0', sizeof(addr));
	addr.sin6_family = AF_INET;
	addr.sin6_flowinfo = 0;
	addr.sin6_port = htons(port);
	addr.sin6_addr = in6addr_any;


	client->socket = socket(addr.sin6_family, SOCK_STREAM, IPPROTO_TCP);
	if (client->socket < 0)
	{
		printf("socket() failed\n");
		_SkyAltClient_socketFree(client);
		return 0;
	}
	if (connect(client->socket, (struct sockaddr *)&addr, sizeof(addr)) < 0)
	{
		printf("connect() failed\n");
		_SkyAltClient_socketFree(client);
		return 0;
	}

	//send UID
	if(!_SkyAltClient_sendPair(client, "uid", uid)){
		return 0;
	}

	//send attributes
	if(!_SkyAltClient_sendAttrs(client)){
		return 0;
	}

	return 1;
}

char SkyAltClient_Get(SkyAltClient* client) {

	int nAttrs = 0;
	{
		char *name = 0;
		if(!_SkyAltClient_recvPairNumber(client, &name, &nAttrs)) {
			return 0;
		}

		char ok = (strcmp(name, "attrs")==0);
		free(name);
		if(!ok) {
			printf("Unknown message type(%s)\n", name);
			return 0;
		}
	}


	for(int i=0; i < nAttrs; i++) {

		SkyAltAttr *attr = 0;
		int nItems = 0;
		{
			char *name = 0;
			if(!_SkyAltClient_recvPairNumber(client, &name, &nItems)) {
				return 0;
			}
			attr = SkyAltClient_FindAttr(client, name);
			if(!attr) {
				printf("Warning: Attribute(%s) not found\n", name);
			}
		}

		for(int ii=0; ii < nItems; ii++) {

			char *name = 0;
			char *value = 0;
			if(!_SkyAltClient_recvPair(client, &name, &value)) {
				return 0;
			}

			if(!attr) {
				continue;
			}

			if(!strcmp(name, "value")) {
				free(attr->Value);
				attr->Value = value;
			} else if(!strcmp(name, "gui_type")) {
				free(attr->Gui_type);
				attr->Gui_type = value;
			} else if(!strcmp(name, "gui_options")) {
				free(attr->Gui_options);
				attr->Gui_options = value;
			} else if(!strcmp(name, "error")) {
				free(attr->Error);
				attr->Error = value;
			} else if(!strcmp(name, "gui_read_only")) {
				attr->Gui_ReadOnly = atoi(value);
				free(value);
			}else{
				free(name);
				free(value);
				printf("Warning: Unknown attribute(%s)\n", name);
			}
		}
	}

	return 1;
}

char SkyAltClient_Progress(SkyAltClient* client, double proc, char* Description, char sendOutputs) {

	if(sendOutputs) {
		if(!_SkyAltClient_sendAttrs(client)){
			return 0;
		}
	}

	//progress
	if(!_SkyAltClient_sendProgress(client, proc, Description, "")){
		return 0;
	}

	return 1;
}

char SkyAltClient_Finalize(SkyAltClient* client) {
	return SkyAltClient_Progress(client, 10, "", 1);
}

char SkyAltClient_Error(SkyAltClient* client, char* err) {

	if(!_SkyAltClient_sendProgress(client, 0, "", err)){
		return 0;
	}

	return 1;
}






//https://github.com/MilanSuk/SkyAlt_c/blob/master/src/os_basic/os_net.h
