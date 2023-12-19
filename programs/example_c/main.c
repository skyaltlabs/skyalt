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

#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <netdb.h>
#include <unistd.h>

#include "skyalt_sdk.h"

int main(int argc, char **argv) {
	if(argc < 3){
		printf("Need 2 arguments: uid, port\n");
		return -1;
	}

	if(!SkyAlt_initGlobal()) {
		return -1;
	}

	SkyAltClient* client = NewSkyAltClient();
	if(client) {
		SkyAltAttr* file = SkyAltClient_AddAttr(client, "file", "");
		SkyAltAttr* query = SkyAltClient_AddAttr(client, "query", "");
		SkyAltAttr* outRows = SkyAltClient_AddAttr(client, "rows", "[]");

		SkyAltClient_Start(client, argv[1], argv[2]);

		while(SkyAltClient_Get(client)) {

			if(!strcmp(file->Value, "")) {
				file->Error = "empty";
				SkyAltClient_Finalize(client);
				continue;
			}
			if(!strcmp(query->Value, "")) {
				query->Error = "empty";
				SkyAltClient_Finalize(client);
				continue;
			}

			//TODO ...

			//result
			outRows->Value = "[{}, {}]";
			if(!SkyAltClient_Finalize(client)) {
				break;
			}
		}

		DeleteSkyAltClient(client);
	}

	SkyAlt_freeGlobal();
}