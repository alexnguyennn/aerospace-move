let windowRule: Type = {
    title: Text,
    titleRegex: Text,
    appRegex: Text,
    app: Text,
}
let space: Type = {
    index: Natural,
    name: Text,
    rules: List windowRule
}
in { spaces: List space }
-- in  {
--     WindowRule = windowRule,
--     Space = space,
-- }


