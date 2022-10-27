from web3 import Web3
from web3.middleware import geth_poa_middleware
from eth_account import Account

web3 = Web3(Web3.HTTPProvider("http://localhost:9650/ext/bc/C/rpc"))
web3.middleware_onion.inject(geth_poa_middleware, layer=0)

privateKey = "0x56289e99c94b6912bfc12adc093c9b51124f0dc54ac7a766b2bc5ccf558d8027"
acct = Account.from_key(privateKey)
web3.eth.defaultAccount = acct
print(web3.eth.get_balance(acct.address))

if len(web3.eth.accounts) == 0:
    print(web3.geth.personal.import_raw_key(privateKey, ""))
    web3.geth.personal.unlock_account(acct.address, '')

print("latest block", web3.eth.block_number, web3.eth.accounts)

web3.eth.send_transaction({
    'to': "0x26Cb836E81bFc47c2530aDBF63968c9830a44C8d",
    'from': acct.address,
    'value': 12345
})

print("latest block", web3.eth.block_number)
