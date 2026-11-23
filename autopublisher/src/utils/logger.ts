// Legacy generation helpers retain redacted diagnostics; provider errors and
// request objects (including headers, URLs and tokens) are never serialized.
export const logger = {
 info:(message:string,_metadata:unknown={})=>console.error(JSON.stringify({level:'info',message})),
 warn:(message:string,_metadata:unknown={})=>console.error(JSON.stringify({level:'warn',message})),
 error:(message:string,_metadata:unknown={})=>console.error(JSON.stringify({level:'error',message}))
};
