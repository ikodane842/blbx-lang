# BLBX error codes

Format: `syntax CODE: message` or `runtime CODE: message`. Each code identifies one category, not an individual occurrence. The same category keeps its code across phases and wrappers. Task errors preserve the underlying code.

The authoritative registry is `syntax/diagnostic/codes.go`. BX4001 is reserved for native failures without a more specific category.

| Code | Meaning |
| --- | --- |
| BX0001 | Source file read failure |
| BX1001 | Unknown source character |
| BX1002 | Unterminated string |
| BX1003 | Unterminated block comment |
| BX1004 | Invalid UTF-8 source |
| BX2001 | Missing or unexpected syntax token |
| BX2002 | Missing or unexpected expression |
| BX2003 | Invalid assignment or destructuring target |
| BX2004 | Syntax nesting limit exceeded |
| BX2005 | Invalid class body |
| BX2006 | Unmatched or misplaced delimiter |
| BX2007 | Missing list separator comma |
| BX2008 | Invalid object field syntax |
| BX2009 | Invalid self binding or parameter |
| BX2010 | Duplicate destructuring binding |
| BX2011 | Rest binding must be last |
| BX2012 | Duplicate destructuring key |
| BX2013 | Duplicate class constructor |
| BX3001 | Undefined variable or function name |
| BX3002 | Undefined object member |
| BX4001 | Unclassified native operation failure |
| BX4002 | Operation unavailable for receiver type |
| BX4003 | Incorrect argument count |
| BX4004 | Incorrect argument or destructuring input type |
| BX4005 | Attempt to call a non-callable value |
| BX4006 | Invalid runtime self usage |
| BX4007 | Class base is not a class |
| BX4008 | Import path or package could not be resolved |
| BX4009 | Requested name absent from imported module |
| BX4010 | Standard input read failure |
| BX4011 | Required destructuring element or field absent |
| BX4012 | Invalid runtime class definition |
| BX4013 | Unexpected task worker failure |
| BX4101 | File or filesystem operation failed |
| BX4102 | Time operation or range failed |
| BX4103 | Network request or response failed |
| BX4104 | Collection operation or range failed |
| BX4105 | Serialization or deserialization failed |
| BX4106 | Math domain or result failure |
| BX4107 | Process or environment operation failed |
| BX4108 | Standard string operation failed |
