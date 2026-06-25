const fs = require('fs');

let path = 'C:/Users/ba876/.gemini/antigravity/scratch/go-proxy/internal/proxy/session.go';
let code = fs.readFileSync(path, 'utf8');

// 1. Add LogDevFee and LogDevFeeError functions
if (!code.includes('func (s *Session) LogDevFee(')) {
    let oldFunc = `func (s *Session) LogGeneral(format string, v ...interface{}) {`;
    let newFunc = `func (s *Session) LogDevFee(format string, v ...interface{}) {
	if s.Config.EnableDetailedLog {
		s.LogGeneral(format, v...)
	}
}

func (s *Session) LogDevFeeError(format string, v ...interface{}) {
	if s.Config.EnableDetailedLog {
		s.LogError(format, v...)
	}
}

func (s *Session) LogGeneral(format string, v ...interface{}) {`;
    code = code.replace(oldFunc, newFunc);
}

// 2. Replace occurrences
code = code.replace(/s\.LogGeneral\("\[SmartRouting\]/g, 's.LogDevFee("[SmartRouting]');
code = code.replace(/s\.LogGeneral\("Connecting to Fee Pool:/g, 's.LogDevFee("Connecting to Fee Pool:');
code = code.replace(/s\.LogGeneral\("\[FEE\] share accepted!/g, 's.LogDevFee("[FEE] share accepted!');
code = code.replace(/s\.LogError\("\[FEE\] share rejected!/g, 's.LogDevFeeError("[FEE] share rejected!');

fs.writeFileSync(path, code);
console.log('Patched session.go successfully!');
