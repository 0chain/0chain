package main

import (
	"strconv"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/threshold/bls"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

var magicBlock = "{\n \"hash\": \"91a6ac5f9de9211a9c30cb82ae7a5dd0ac16f35b152af7601c6fea39ae222bd8\",\n \"previous_hash\": \"\",\n \"magic_block_number\": 1,\n \"starting_round\": 0,\n \"miners\": {\n  \"type\": 0,\n  \"nodes\": {\n   \"166d8c24bc6cf2fbea5ffcc2f3e5c97d9773817a60fb73afc0ecfc03a28b9747\": {\n    \"id\": \"166d8c24bc6cf2fbea5ffcc2f3e5c97d9773817a60fb73afc0ecfc03a28b9747\",\n    \"version\": \"\",\n    \"creation_date\": 1663087960,\n    \"public_key\": \"19f240532dc618b28ebd75ff41ded36490b8e92832f1b7180ec230c920244e1eb084e9feae7c66d74fdc2c79cd9465b770712a53dd886a480c8c4ec55f53d918\",\n    \"n2n_host\": \"as1.testnet-0chain.net\",\n    \"host\": \"as1.testnet-0chain.net\",\n    \"port\": 7071,\n    \"path\": \"miner01\",\n    \"type\": 0,\n    \"description\": \"as1@0chain.com\",\n    \"set_index\": 0,\n    \"status\": 0,\n    \"in_prev_mb\": false,\n    \"info\": {\n     \"build_tag\": \"\",\n     \"state_missing_nodes\": 0,\n     \"miners_median_network_time\": 0,\n     \"avg_block_txns\": 0\n    }\n   },\n   \"2aeb9cce14ba34253e39c9a8b8afa97472928dd68f2f5a45dc108969ee09635b\": {\n    \"id\": \"2aeb9cce14ba34253e39c9a8b8afa97472928dd68f2f5a45dc108969ee09635b\",\n    \"version\": \"\",\n    \"creation_date\": 1663087960,\n    \"public_key\": \"ac0f6bc73fb1697632718ac03e59b13ded1ecb3463ba23e8f531572f92d50522c347087bfc99e82f2b7b3aeb9e9d2064588f6ce378ada81c73e0c4737ac262a0\",\n    \"n2n_host\": \"as3.testnet-0chain.net\",\n    \"host\": \"as3.testnet-0chain.net\",\n    \"port\": 7071,\n    \"path\": \"miner01\",\n    \"type\": 0,\n    \"description\": \"as3@0chain.net\",\n    \"set_index\": 1,\n    \"status\": 0,\n    \"in_prev_mb\": false,\n    \"info\": {\n     \"build_tag\": \"\",\n     \"state_missing_nodes\": 0,\n     \"miners_median_network_time\": 0,\n     \"avg_block_txns\": 0\n    }\n   },\n   \"4a3c6604231c3d0497c65aa3ca0287a164f169e7e6ea9863a3ef8203e987f1ee\": {\n    \"id\": \"4a3c6604231c3d0497c65aa3ca0287a164f169e7e6ea9863a3ef8203e987f1ee\",\n    \"version\": \"\",\n    \"creation_date\": 1663087960,\n    \"public_key\": \"c60709f10b96447985464f3b4f441ba0ff093887a7eec751184ae9359390eb000ce56a8b13c386b362b37e7266dfca65865a6f6f6e123e1f6ef7b21b1aa15f88\",\n    \"n2n_host\": \"as2.testnet-0chain.net\",\n    \"host\": \"as2.testnet-0chain.net\",\n    \"port\": 7071,\n    \"path\": \"miner01\",\n    \"type\": 0,\n    \"description\": \"as2@0chain.com\",\n    \"set_index\": 2,\n    \"status\": 0,\n    \"in_prev_mb\": false,\n    \"info\": {\n     \"build_tag\": \"\",\n     \"state_missing_nodes\": 0,\n     \"miners_median_network_time\": 0,\n     \"avg_block_txns\": 0\n    }\n   }\n  }\n },\n \"sharders\": {\n  \"type\": 1,\n  \"nodes\": {\n   \"af89361927f0261fa7f4c2774ca49290a05b5f38d8f91278f395ca7b6c863178\": {\n    \"id\": \"af89361927f0261fa7f4c2774ca49290a05b5f38d8f91278f395ca7b6c863178\",\n    \"version\": \"\",\n    \"creation_date\": 1663087960,\n    \"public_key\": \"e568d52071398b2b6c8cdbb1bf0e01a4c52eeaa7c77c5484372258a904204900a48639a1b732f3008a4af48fda6064428af5101ba949e938d42b9685333c4299\",\n    \"n2n_host\": \"as3.testnet-0chain.net\",\n    \"host\": \"as3.testnet-0chain.net\",\n    \"port\": 7171,\n    \"path\": \"sharder01\",\n    \"type\": 1,\n    \"description\": \"as3@0chain.net\",\n    \"set_index\": 0,\n    \"status\": 0,\n    \"in_prev_mb\": false,\n    \"info\": {\n     \"build_tag\": \"\",\n     \"state_missing_nodes\": 0,\n     \"miners_median_network_time\": 0,\n     \"avg_block_txns\": 0\n    }\n   },\n   \"b3ca9075e4a3c312dce345c992f09e43f408a2e96d57965c0aa8557be8ce9473\": {\n    \"id\": \"b3ca9075e4a3c312dce345c992f09e43f408a2e96d57965c0aa8557be8ce9473\",\n    \"version\": \"\",\n    \"creation_date\": 1663087960,\n    \"public_key\": \"56cdb0471e47e1bfa375ad938af93259a7474f3358376da579cbab15b1652604b85063655f85ac5c5f8b6f8807ce6f8f9160696ddaeb0001f77f1a27826ce21e\",\n    \"n2n_host\": \"as2.testnet-0chain.net\",\n    \"host\": \"as2.testnet-0chain.net\",\n    \"port\": 7171,\n    \"path\": \"sharder01\",\n    \"type\": 1,\n    \"description\": \"as2@0chain.com\",\n    \"set_index\": 1,\n    \"status\": 0,\n    \"in_prev_mb\": false,\n    \"info\": {\n     \"build_tag\": \"\",\n     \"state_missing_nodes\": 0,\n     \"miners_median_network_time\": 0,\n     \"avg_block_txns\": 0\n    }\n   },\n   \"dcdf6135a5af6cc13db6a693b53f508040cf6fa12895883d87a36a047dc22033\": {\n    \"id\": \"dcdf6135a5af6cc13db6a693b53f508040cf6fa12895883d87a36a047dc22033\",\n    \"version\": \"\",\n    \"creation_date\": 1663087960,\n    \"public_key\": \"7d8f7642928a9c4c464e002ee94074180ebbd804175d9282983205ee396bd821977f509aed484c9b547d19be0df4ce061cc46f8be980f9f4f68afd1dd6aa4593\",\n    \"n2n_host\": \"as1.testnet-0chain.net\",\n    \"host\": \"as1.testnet-0chain.net\",\n    \"port\": 7171,\n    \"path\": \"sharder01\",\n    \"type\": 1,\n    \"description\": \"as1@0chain.com\",\n    \"set_index\": 2,\n    \"status\": 0,\n    \"in_prev_mb\": false,\n    \"info\": {\n     \"build_tag\": \"\",\n     \"state_missing_nodes\": 0,\n     \"miners_median_network_time\": 0,\n     \"avg_block_txns\": 0\n    }\n   }\n  }\n },\n \"share_or_signs\": {\n  \"shares\": {\n   \"166d8c24bc6cf2fbea5ffcc2f3e5c97d9773817a60fb73afc0ecfc03a28b9747\": {\n    \"id\": \"166d8c24bc6cf2fbea5ffcc2f3e5c97d9773817a60fb73afc0ecfc03a28b9747\",\n    \"share_or_sign\": {\n     \"2aeb9cce14ba34253e39c9a8b8afa97472928dd68f2f5a45dc108969ee09635b\": {\n      \"id\": \"\",\n      \"message\": \"af2bf61ef121ad85bd1c119625c83245db062a84aeebbb089f2c33c36e15da56\",\n      \"share\": \"\",\n      \"sign\": \"1 14f41784e04b613828b71751c5350f1ee887ebd75f0fe12313afd933aea6a06d 71ec6b605e4ed7cfc084ac3312afe408a66573ff4210989d1d054d8f15c7721\"\n     },\n     \"4a3c6604231c3d0497c65aa3ca0287a164f169e7e6ea9863a3ef8203e987f1ee\": {\n      \"id\": \"\",\n      \"message\": \"1c916b627489f92d76d8c59870e654336d7a48e3d3343ae4f3710193c906b7d3\",\n      \"share\": \"\",\n      \"sign\": \"1 cdde0fe45951420f66d27e1c601a20a62dc88b9ecf890c975a4cf383978610f 13b02f0f9b74be1c4f62df30c16c724d32838a3c9294a1f0413d34da224ba29c\"\n     }\n    }\n   },\n   \"2aeb9cce14ba34253e39c9a8b8afa97472928dd68f2f5a45dc108969ee09635b\": {\n    \"id\": \"2aeb9cce14ba34253e39c9a8b8afa97472928dd68f2f5a45dc108969ee09635b\",\n    \"share_or_sign\": {\n     \"166d8c24bc6cf2fbea5ffcc2f3e5c97d9773817a60fb73afc0ecfc03a28b9747\": {\n      \"id\": \"\",\n      \"message\": \"aeef3d41fa1eccac01e0862d35fa82f2dcb79356037c9a607e8c6ef69e2ebaed\",\n      \"share\": \"\",\n      \"sign\": \"1 6b93432606d5b37ef79cd1c0fff4cce916f14e9307734f80036e634d195525a 199498deed6579c5b602492584657dccff32b6c0098dda999e23b0332ef231c\"\n     },\n     \"4a3c6604231c3d0497c65aa3ca0287a164f169e7e6ea9863a3ef8203e987f1ee\": {\n      \"id\": \"\",\n      \"message\": \"bc8166335d19b4f3d12c48d117f38afcf9ce0ba3dcd1b265eb4c463cdfceb76a\",\n      \"share\": \"\",\n      \"sign\": \"1 18a9d7bbfe28a2559b6e3ac3170e6c4674f4ff4b1eabbff0bdf9ba6e4d3a7ed7 8bb83d7fa8ea85a88d9379821d3e78395b828c48f1cf689245a349c7e2ace58\"\n     }\n    }\n   },\n   \"4a3c6604231c3d0497c65aa3ca0287a164f169e7e6ea9863a3ef8203e987f1ee\": {\n    \"id\": \"4a3c6604231c3d0497c65aa3ca0287a164f169e7e6ea9863a3ef8203e987f1ee\",\n    \"share_or_sign\": {\n     \"166d8c24bc6cf2fbea5ffcc2f3e5c97d9773817a60fb73afc0ecfc03a28b9747\": {\n      \"id\": \"\",\n      \"message\": \"0ccb9202371e17ac9c93ba99b009b92f9a5771823646807136b8cc7ee7831bee\",\n      \"share\": \"\",\n      \"sign\": \"1 d45d78900012695183529b7315e83e2b81511385a1fe9e9dee8c437729fe3ed 1a7fc57480678516d06b6fb4e9747b3ab8cf556d85bbc074319e2392d822e972\"\n     },\n     \"2aeb9cce14ba34253e39c9a8b8afa97472928dd68f2f5a45dc108969ee09635b\": {\n      \"id\": \"\",\n      \"message\": \"c948adc05c76fe3ed27d02d0499e4ee290c62e3dfc2f71aafcdce267184783ff\",\n      \"share\": \"\",\n      \"sign\": \"1 150812953fd0ac3ffd4467210ba76c41d900493cfab29c066a95374948cb8fe 21f68e8300186afed297067bde6ee24e3f85c29acd7d5246716a886ff55e51d1\"\n     }\n    }\n   }\n  }\n },\n \"mpks\": {\n  \"Mpks\": {\n   \"166d8c24bc6cf2fbea5ffcc2f3e5c97d9773817a60fb73afc0ecfc03a28b9747\": {\n    \"ID\": \"166d8c24bc6cf2fbea5ffcc2f3e5c97d9773817a60fb73afc0ecfc03a28b9747\",\n    \"Mpk\": [\n     \"1 188b365f0069998cb8c3b996eaaa64aeddb436da2cb0547ac1b066844650cf5d a227f85b543ff8c4081c4d0253bf10ecfb09fa8a69cfb05e3e3095b57ebd9a3 bdaf3bd74ca12bfc539ec5dc04f312b2b45b6e17b47b3ff1fed71a1928cf6aa d8a415b8fed8acc2dd9378f4b9314c0e07804cc4c8301e5060c020f5f610d56\",\n     \"1 18dc829801285f09e580aec1b4800fe3e3113329f719a034aa869a7e53bc28bb 9666f98c6e4fa8e0db1acc031aa9998536df98eb30aa0eac66abbd034caf113 16f29b47608952b634262eb93d780ef4b1dc40f90208c1963be42bfecd8b66b 176e4a92e5b8e0f00286deab092d9a5fb111eba97141a9aed12c3132dc65a752\",\n     \"1 8fad3bebdd1a85103958b08d1dde22ab031f7d8e2707cb988fb2ea5c45db8b2 d0a2bfde40dc3c524a4aa22a7e505e0b6527818fd82b216825fb82efdb144bc 7c75241f3105c62052442a4ba7f869e9c8e3519d1305de333f1b8bcf4b436d4 85dbfbd6eae4a4b96c3e587da4cba950fd6c03594db8589ccca506fa2c5b2f9\"\n    ]\n   },\n   \"2aeb9cce14ba34253e39c9a8b8afa97472928dd68f2f5a45dc108969ee09635b\": {\n    \"ID\": \"2aeb9cce14ba34253e39c9a8b8afa97472928dd68f2f5a45dc108969ee09635b\",\n    \"Mpk\": [\n     \"1 23a1e071857c54c57c65b981e4fb827716de8518153eec9f673021268404be6 4e0aa9e31565ac67247e4f6ff76add9ea892d34a02d14bead7fea74d24cc6dc 7f4e7c71a7242b397ebd39833f28c8d9f78a6da3aecfcb0a024b7d84d5db7eb 38273f6643af2329c5682213d6fabb2028f28ad0f15db91dbfff138624f490c\",\n     \"1 1c26237483f43f7bc5fa4dadc8a728e9c8cc0e4692902ebf80fc5b3d36a8ec12 10a95f746b35dd582288a03e1931d0f3620d65d488d07545cedc1424e66d0584 42751b79a3b5420207826763393f40f80c9988399853603ca3a9e3aeba201d1 9e49a00af0201e8fae8b6d403fdc8f5ee6de2ca6767bde490d585975f964609\",\n     \"1 21640624bac26dc9ae07026ad394b6c9295866b689944e1cc3bc7a2354ca852b 85602e4c1dc4bd02956aa3a0d2055f8cedd620024ac836a23cb8f2c6a6962ff 1ff3fbf684e02ca59b6c7700a2eaad353fa75070ce3c48410d18200154db8610 64495e6f7785f92f286a92894ba63ee73da430d57dc4f087424b7743a16cea6\"\n    ]\n   },\n   \"4a3c6604231c3d0497c65aa3ca0287a164f169e7e6ea9863a3ef8203e987f1ee\": {\n    \"ID\": \"4a3c6604231c3d0497c65aa3ca0287a164f169e7e6ea9863a3ef8203e987f1ee\",\n    \"Mpk\": [\n     \"1 183643ea7cebb15e6d2dca3bc1faf447b3bec5875a9955db194788c64b443162 59db0552aab80e1474022acc372d48310efe1563839ac2e1ad7e02485bf5d0c e960ac1c81355bc31210679be48fc430863f14e7e8a88bc63a89b2ce15926c0 31840c9ea3ce093f6f034b66e2a5cfcd7798bc341cd24f4ab000787797869f9\",\n     \"1 8d3accc2588914109593175cd24791fbd833700cdf0c7e4cbbbeff42f335ba5 1004b9382ad7408aa952969ec4ef5c98228da34794739eeb764e85a3b60e4a17 2137255264a4b7623b56fe6426ba29ebf75117fd4709ab9db9891ba36efc80ec 1bcb309cd57cf0cb804fbc6d6a8c86f1f96eb1ef70047750271eea2cfaf43630\",\n     \"1 1be8f0ffdd42f675128d6501543014be19c68f2e82f14537a224f4e904e6d4f7 8ba18134023802dabb1fc3846dd4041fc10f6d5b8b3fdc37146fd1e75d63241 2413fe77c62bbb463a0c844e64f781a0a613fb824bba33bef3661567b92541b0 10753fcc7794ed1a7e68cf5bc3b00427a9e7f6925910bdaec0c7ceed4d8f8a2f\"\n    ]\n   }\n  }\n },\n \"t\": 3,\n \"k\": 3,\n \"n\": 3\n}"
var dkg1 = "{\n \"id\": \"1\",\n \"starting_round\": 0,\n \"secret_shares\": {\n  \"1166d8c24bc6cf2fbea5ffcc2f3e5c97\": \"1bcf1fccba745cf871ba591c70b09336e795e5ad7d01350649ef251f48426545\",\n  \"12aeb9cce14ba34253e39c9a8b8afa97\": \"1b9191a87ac8707f9e60262dc97a178060c65e1cf936463e806502086c2c18b9\",\n  \"14a3c6604231c3d0497c65aa3ca0287a\": \"af3e4e8bd8cf5a6620d67ca4896966b52b417fad3bdf0a743a14062dab7d30d\"\n }\n}"
var dkg2 = "{\n \"id\": \"1\",\n \"starting_round\": 0,\n \"secret_shares\": {\n  \"1166d8c24bc6cf2fbea5ffcc2f3e5c97\": \"159876b061ec2f89a30b9ae69dd70c0abd7e122030947ce617b937541cc869ac\",\n  \"12aeb9cce14ba34253e39c9a8b8afa97\": \"12241e07d103013e16856307c18517b59240113ff6a42fcc7752405152e95ea4\",\n  \"14a3c6604231c3d0497c65aa3ca0287a\": \"1f08fb40d9d5eae394f91cb9e3a009274d93e198515b05362e215c3143432736\"\n }\n}"
var dkg3 = "{\n \"id\": \"1\",\n \"starting_round\": 0,\n \"secret_shares\": {\n  \"1166d8c24bc6cf2fbea5ffcc2f3e5c97\": \"25f654bd382cfefdd53e11ed90b48fd820668c1c5ac8adc71de3f801323c1e9\",\n  \"12aeb9cce14ba34253e39c9a8b8afa97\": \"d8ba2fdddcf51931b0fb1160e5594e4ac27bf0f04ea8780c5bf4bce301ad62f\",\n  \"14a3c6604231c3d0497c65aa3ca0287a\": \"675d0ca16c4ea1630461e40b794451d8f47a644487580a0db6eeed5a7a057ad\"\n }\n}"
var miner1 = "166d8c24bc6cf2fbea5ffcc2f3e5c97d9773817a60fb73afc0ecfc03a28b9747"
var miner2 = "2aeb9cce14ba34253e39c9a8b8afa97472928dd68f2f5a45dc108969ee09635b"
var miner3 = "4a3c6604231c3d0497c65aa3ca0287a164f169e7e6ea9863a3ef8203e987f1ee"

//func TestKeyGeneration(t *testing.T) {
//	hexString := "1234c153219f3688b8715670dd9d28d54e93f2c44bf65d0036c604a199a7a623"
//
//	pk, pub := GenerateKeys()
//	log.Println(pk, pub)
//	var privateKey bls.SecretKey
//	if err := privateKey.SetHexString(hexString); err != nil {
//		log.Panic(err)
//	}
//
//}

func TestMagicBlockValidity(t *testing.T) {
	var err error
	mb := block.NewMagicBlock()
	if err = mb.Decode([]byte(magicBlock)); err != nil {
		t.Error(err)
	}

	var (
		dkgShare = &bls.DKGSummary{
			SecretShares: make(map[string]string),
		}
	)

	dkgShare = &bls.DKGSummary{SecretShares: make(map[string]string)}
	if err = dkgShare.Decode([]byte(dkg3)); err != nil {
		t.Error(err)
	}

	dkgShare.ID = strconv.FormatInt(mb.MagicBlockNumber, 10)

	mpks, err := mb.Mpks.GetMpkMap()
	if err != nil {
		logging.Logger.Panic("Get mpks map failed", zap.Error(err))
	}

	if err = dkgShare.Verify(bls.ComputeIDdkg(miner3), mpks); err != nil {
		t.Error(err)
	}

}

////GenerateKeys - implement interface
//func GenerateKeys() (string, string) {
//	var skey bls.SecretKey
//	skey.SetByCSPRNG()
//	pub := skey.GetPublicKey().SerializeToHexStr()
//	return skey.GetHexString(), pub
//}
