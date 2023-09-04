
-- { 
--   spaces =
--   [ { index = 0
--     , name = "1"
--     , window_rules =
--       [ { app = "", app_regex = "", title = "", title_regex = "" } ]
--     }
--   ]
-- }
-- let 
--   x = [1, 2, 3 ]
-- in
-- {
--   foo = x,
--   bar = True,
--   new = [
--     {
--       x = 1,
--       y = "ABC",
--     }
--   ]
-- } : { foo: List Natural, bar: Bool, new: List { x: Natural, y : Text } }

-- -- Import packages
-- let 
--   JSON =
--         https://prelude.dhall-lang.org/v19.0.0/JSON/package.dhall sha256:79dfc281a05bc7b78f927e0da0c274ee5709b1c55c9e5f59499cb28e9d6f3ec0

-- let testOutput : Natural = 1 in assert: JSON.render (JSON.object [{mapKey = "x", mapValue = JSON.string "1"}]) 
--     === 
--     ''
--     { "x": "1" }
--     ''

-- NOTE: this seems finicky to test, RHS needs to be exactly right space format since it's just a big string
-- need better way to compare
-- string -> parse to dhall --> compare object instead?

-- let State : Type = <NY | TX>

-- let Address: Type = {
--   state: State,
--   street: Text,
--   city: Text,
--   number: Text,
-- }

-- let Person: Type = {
--   name: Text,
--   address: Address,
-- }

-- let john = { name = "John Doe", address = { city = "Austin", state = State.TX, number = "9999", street = "Main Street" } : Address }: Person
-- let jane : Person = { name = "Janet Doet", address = { city = "New York", state = State.NY, number = "9999", street = "Main Street" } : Address  } 

-- in [
--   john,
--   jane
-- ] : List Person

let smallServer =
      \(hostName : Text) ->
        { cpus = 1
        , gigabytesOfRAM = 1
        , hostName = hostName
        , terabytesOfDisk = 1
        }

let mediumServer =
      \(hostName : Text) ->
        { cpus = 8
        , gigabytesOfRAM = 16
        , hostName = hostName
        , terabytesOfDisk = 4
        }

let largeServer =
      \(hostName : Text) ->
        { cpus = 64
        , gigabytesOfRAM = 256
        , hostName = hostName
        , terabytesOfDisk = 16
        }

in  [ smallServer "eu-west.example.com"
    , largeServer "us-east.example.com"
    , largeServer "ap-northeast.example.com"
    , mediumServer "us-west.example.com"
    , smallServer "sa-east.example.com"
    , largeServer "ca-central.example.com"
    ]