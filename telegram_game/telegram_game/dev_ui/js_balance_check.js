const fs = require("fs");

const file = process.argv[2] || ".\\.check_scripts\\script_01.js";
const src = fs.readFileSync(file, "utf8");

let line = 1, col = 0;
let mode = "normal"; // normal | squote | dquote | template | linec | blockc
const stack = [];    // {ch,line,col,tag}

function push(ch, tag="") { stack.push({ ch, line, col, tag }); }
function popExpect(closeCh){
  if(stack.length === 0){
    console.error(`EXTRA closing "${closeCh}" at ${line}:${col}`);
    process.exit(2);
  }
  const top = stack[stack.length-1];
  const ok =
    (top.ch === "{" && closeCh === "}") ||
    (top.ch === "(" && closeCh === ")") ||
    (top.ch === "[" && closeCh === "]");
  if(!ok){
    console.error(`MISMATCH closing "${closeCh}" at ${line}:${col} (top "${top.ch}" opened at ${top.line}:${top.col})`);
    process.exit(3);
  }
  const popped = stack.pop();
  // template expr kapanınca template moduna geri dön
  if(popped.tag === "tpl" && closeCh === "}") mode = "template";
}

for(let i=0; i<src.length; i++){
  const c = src[i];
  const n = src[i+1];

  if(c === "\n"){
    line++; col = 0;
    if(mode === "linec") mode = "normal";
    continue;
  }
  col++;

  // --- string/comment modları ---
  if(mode === "linec") continue;

  if(mode === "blockc"){
    if(c === "*" && n === "/"){ mode = "normal"; i++; col++; }
    continue;
  }
  if(mode === "squote"){
    if(c === "\\"){ i++; col++; continue; }
    if(c === "'") mode = "normal";
    continue;
  }
  if(mode === "dquote"){
    if(c === "\\"){ i++; col++; continue; }
    if(c === '"') mode = "normal";
    continue;
  }
  if(mode === "template"){
    if(c === "\\"){ i++; col++; continue; }
    if(c === "`"){ mode = "normal"; continue; }
    if(c === "$" && n === "{"){
      // ${ ... } -> expr başlangıcı: özel tag ile { push
      push("{","tpl");
      mode = "normal";
      i++; col++;
      continue;
    }
    continue;
  }

  // --- normal mod ---
  if(c === "/" && n === "/"){ mode = "linec"; i++; col++; continue; }
  if(c === "/" && n === "*"){ mode = "blockc"; i++; col++; continue; }
  if(c === "'"){ mode = "squote"; continue; }
  if(c === '"'){ mode = "dquote"; continue; }
  if(c === "`"){ mode = "template"; continue; }

  if(c === "{" || c === "(" || c === "["){ push(c); continue; }
  if(c === "}" || c === ")" || c === "]"){ popExpect(c); continue; }
}

if(mode !== "normal"){
  console.error(`UNCLOSED MODE at EOF: ${mode}`);
  process.exit(1);
}

if(stack.length){
  console.error("UNCLOSED TOKENS at EOF (last 12):");
  stack.slice(-12).forEach(s=>{
    console.error(`  ${s.ch} opened at ${s.line}:${s.col}${s.tag ? " ["+s.tag+"]" : ""}`);
  });
  process.exit(1);
}

console.log("OK (heuristic): no obvious unclosed braces/strings/comments.");
