# YeahWooGo

YeahWooGo is a tool designed to simplify the development of backend applications in Go for programmers. Our goal is to craft an experience so intuitive and efficient that it evokes exclamations of "Yeah!", "Woo!", and "Go!" from users.

## Background

In backend development, we frequently face scenarios such as:
- Encountering an unexpected final value in a variable and needing to trace its assignments to identify the anomaly.
- Determining the appropriate place to integrate new logic that will change a variable's value for feature enhancement.
- Refactoring code by converting repeated routines into functions to streamline and clarify variable lifecycle management.

In these instances, it's crucial to focus solely on the segments of code concerning the variable in question. YeahWooGo is designed to assist developers in this task by automating the process. To start, we've crafted a feature that tackles this process within the scope of single functions.

## Main Process

We would like to outline the primary steps of our algorithm:
- First, parse the file to extract the Abstract Syntax Tree (AST) of a given function.
- Next, transform the function's AST into a universally defined structure that represents any type of AST node.
- Starting with the initial function node, recursively filter each node and its child nodes through the following criteria:
    - If a node is a variable name, verify whether it matches the target variable.
    - If a node is an *ast.BlockStmt, retain only the statements that are relevant alongside all return statements, discarding the rest.
- Upon completing the filtration, we obtain a refined AST of the function, which we then output.

To clarify, we use two key terms within our algorithm:

A node is considered the "target variable" if it corresponds to a variable name and:

- For an unqualified name, represented by *ast.Ident, it is an exact match with the target variable's name.
- For a qualified name, denoted by *ast.SelectorExpr, the node is the target if it is either a substring of the target variable's name, or if the target variable's name is a substring of the node.

A node is considered "relevant" if it either represents the target variable or contains any relevant sub-node.

## Usage

Our tool accepts input in JSON format. The initial parameter provided to the executable is the file path of the JSON input. Here’s a sample of the expected input structure:

```json
{
    "method": "GetRelevantFunc", 
    "source": "path/to/your/file.go",
    "func_name": "YourFunctionName",
    "var_name": "VariableName"
}
```

# License

This work is licensed under a Creative Commons Attribution-NonCommercial-NoDerivatives 4.0 International License.
