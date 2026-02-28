import { execSync } from 'node:child_process';
import pico from 'picocolors';

const checks = [
  {
    name: 'Go Toolchain',
    command: 'go version',
    match: /go version go1\.(\d+)/,
    validate: (match) => {
      if (!match) return 'Go is not installed or version unknown.';
      const minor = parseInt(match[1], 10);
      if (minor < 22) return `Go 1.22+ required, found 1.${minor}`;
      return true;
    }
  },
  {
    name: 'C Compiler (GCC/Clang)',
    command: 'gcc --version',
    fallback: 'clang --version',
    validate: () => true, // Just needs to exist
  },
  {
    name: 'Make',
    command: 'make --version',
    validate: () => true,
  },
  {
    name: 'Node.js',
    command: 'node --version',
    match: /v(\d+)/,
    validate: (match) => {
      if (!match) return 'Node is not installed.';
      const major = parseInt(match[1], 10);
      if (major < 20) return `Node 20+ required, found ${major}`;
      return true;
    }
  }
];

function runCommand(cmd) {
  try {
    return execSync(cmd, { stdio: 'pipe', encoding: 'utf-8' }).trim();
  } catch (e) {
    return null;
  }
}

console.log(pico.bold(pico.cyan('\n🩺 Neuron CLI Doctor\n')));

let allPassed = true;

for (const check of checks) {
  process.stdout.write(`Checking ${check.name}... `);
  
  let output = runCommand(check.command);
  if (!output && check.fallback) {
    output = runCommand(check.fallback);
  }

  if (!output) {
    console.log(pico.red('❌ Missing'));
    if (check.help) console.log(`   └─ ${pico.gray(check.help)}`);
    allPassed = false;
    continue;
  }

  if (check.match && check.validate) {
    const match = output.match(check.match);
    const result = check.validate(match);
    if (result !== true) {
      console.log(pico.red('❌ Failed'));
      console.log(`   └─ ${pico.yellow(result)}`);
      allPassed = false;
      continue;
    }
  }

  const firstLine = output.split('\n')[0].substring(0, 40);
  console.log(pico.green('✅ OK ') + pico.gray(`(${firstLine}...)`));
}

console.log('\n----------------------------------------');
if (allPassed) {
  console.log(pico.bold(pico.green('🎉 All dependencies look good! You are ready to build neuron.')));
} else {
  console.log(pico.bold(pico.red('⚠️  Some checks failed. Please install missing dependencies (see SETUP.md).')));
  process.exit(1);
}
console.log();
