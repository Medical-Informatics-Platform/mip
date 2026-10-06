# Jupyter Notebooks and Cohort Scout

MIP now offers a JupyterLab workspace alongside the web analysis workflow.
Notebooks combine Python code, analysis results, visualizations, and explanatory
text so researchers can develop and revisit reproducible federated analyses.

## Getting started

1. Open the notebook entry point in the MIP portal. Notebook access requires
   JupyterHub to be configured for the installation.
2. Open `Welcome.ipynb` and run its cells to verify the platform connection.
3. Browse the supplied examples and copy an example into `scratch/` to adapt it.
4. Discover data models, datasets, and variables, build an analysis selection,
   apply filters and preprocessing, and run algorithms with the `mip` client.
5. Save personal notebooks and analysis scripts in `scratch/`.

The workspace includes a pre-installed Python client and session configuration:

```python
import mip

client = mip.Client.from_env()
catalog = client.catalog()
```

The client exposes analysis selections and pipelines for filters, missing-value
handling, outlier treatment, and reusable K-means cluster columns. Notebook
requests go through the platform backend and its access controls to Exaflow;
patient-level records remain at the contributing sites.

## Notebook support agent

**Cohort Scout** is the MIP notebook support agent available through Jupyter AI
when the assistant service is configured. It helps researchers create and edit
notebooks, understand existing code and analysis results, discover available
analyses, and use the MIP Python client.

Example requests include:

- “Create a notebook that explores the available datasets and runs a histogram.”
- “Explain the preprocessing steps and algorithm used in this notebook.”
- “Help me adapt an example analysis to my selected variables.”

## Repository and further reading

The new [mip-jupyter repository](https://github.com/Medical-Informatics-Platform/mip-jupyter/tree/0.1.1)
contains the workspace template, Python client, example notebooks, user guides,
Jupyter images, and assistant integration.

- [Quickstart](https://github.com/Medical-Informatics-Platform/mip-jupyter/blob/0.1.1/docs/user/quickstart.md)
- [Python client API reference](https://github.com/Medical-Informatics-Platform/mip-jupyter/blob/0.1.1/docs/user/api-reference.md)
- [Workspace guide](https://github.com/Medical-Informatics-Platform/mip-jupyter/blob/0.1.1/docs/user/workspace-guide.md)
- [Operator integration guide](https://github.com/Medical-Informatics-Platform/mip-jupyter/blob/0.1.1/docs/operators.md)
