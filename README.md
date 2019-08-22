# Financial accounting

## Environment

* SBERBANK_ENTRYPOINT_URL `entry point for sberbank api https://3dsec.sberbank.ru`
* SBERBANK_TOKEN `token for sberbank api`
* SBERBANK_USER_NAME `user name for sberbank api`
* SBERBANK_PASSWORD `password for sberbank api`
* MOEDELO_ENTRYPOINT_URL `entry point for moe delo api https://restapi.moedelo.org`
* MOEDELO_TOKEN `toke for moe delo api`

## cli

Create token: `./bin/acca-race --gen-access-token`


Result: 

```json
{"access_token":"3f6a8e504ef5f59f6537f1ac7215e8793c55d48fe9a08fc0f5ac778e12680307","currentcy":"rub","system_account_id":1}
```
