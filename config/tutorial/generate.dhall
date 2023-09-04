-- header block comment
let windowRuleType: Type = { title : Text, titleRegex : Text, appRegex : Text, app : Text }
let windowRule = { Type = windowRuleType 
      , default = { title = "", titleRegex = "", app = "", appRegex = "" }
      }

let space
    : Type
    = { index : Natural, name : Text, rules : List windowRuleType }

-- in {
--     spaces: List space
-- }
in  { 
    -- something
    rules= [
    windowRule::{ titleRegex = "firefox"}
] : List windowRuleType}
