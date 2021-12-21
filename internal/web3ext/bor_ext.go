package web3ext

// BorJs bor related apis
const BorJs = `
web3._extend({
	property: 'bor',
	methods: [
		new web3._extend.Method({
			name: 'getSnapshot',
			call: 'bor_getSnapshot',
			params: 1,
			inputFormatter: [null]
		}),
		new web3._extend.Method({
			name: 'test',
			call: 'bor_test',
			params: 0,
			inputFormatter: [null]
		}),
	]
});
`
