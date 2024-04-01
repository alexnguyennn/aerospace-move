
schema WindowRule:
    """
    Some docstring

    Attributes
    ----------
    title: description of window to match

    """
    # title description
    title: str
    titleRegex: str # regex describing title
    appRegex: str
    """ app description """
    app: str
        

schema WorkspaceConfig:
    index: int = 0
    name: str = "default"
    rules: [WindowRule] = [WindowRule{
        title = "test"
        titleRegex = ""
        app = ""
        appRegex = ""
    }]
