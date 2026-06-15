# Federated Analysis Algorithms

The MIP federated analysis algorithms are implemented and documented in Exaflow.
The algorithm-specific technical documentation is available in the Exaflow
repository under
[documentation/algorithms](https://github.com/madgik/exaflow/tree/master/documentation/algorithms).

The linked Exaflow documentation describes each algorithm's inputs, parameters,
statistical or machine-learning method, federated computation, outputs,
validation reference, and limitations.

## Summary and Visualization

| Algorithm | Description | Documentation |
|---|---|---|
| Descriptive Statistics | Summarizes selected numerical and nominal variables with per-dataset and combined summaries. | [Documentation](https://github.com/madgik/exaflow/blob/master/documentation/algorithms/Describe.md) |
| Histogram | Computes counts for one numerical or categorical variable, optionally split by categorical grouping variables. | [Documentation](https://github.com/madgik/exaflow/blob/master/documentation/algorithms/Histogram.md) |
| Outlier Report | Reports outlier bounds, counts, and percentages for numerical variables using Gaussian, IQR, MAD, or quantile rules. | [Documentation](https://github.com/madgik/exaflow/blob/master/documentation/algorithms/outlier_report.md) |

## Statistical Tests

| Algorithm | Description | Documentation |
|---|---|---|
| One-way ANOVA | Tests whether a numerical outcome has the same mean across levels of one categorical grouping variable. | [Documentation](https://github.com/madgik/exaflow/blob/master/documentation/algorithms/ANOVAOneWay.md) |
| Two-way ANOVA | Tests whether a numerical outcome differs across two categorical factors, including main effects and interaction. | [Documentation](https://github.com/madgik/exaflow/blob/master/documentation/algorithms/ANOVATwoWay.md) |
| Chi-squared Test | Evaluates whether two categorical variables are independent using a contingency table. | [Documentation](https://github.com/madgik/exaflow/blob/master/documentation/algorithms/ChiSquared.md) |
| Fisher's Exact Test | Evaluates association between two binary categorical variables using exact 2 by 2 table probabilities. | [Documentation](https://github.com/madgik/exaflow/blob/master/documentation/algorithms/FisherExact.md) |
| Independent t-test | Compares the mean of a numerical variable between two independent groups using the pooled-variance Student t-test. | [Documentation](https://github.com/madgik/exaflow/blob/master/documentation/algorithms/TtestIndependent.md) |
| One-sample t-test | Compares the mean of a numerical variable with a reference mean. | [Documentation](https://github.com/madgik/exaflow/blob/master/documentation/algorithms/TtestOneSample.md) |
| Paired t-test | Compares two related numerical measurements by testing whether the mean paired difference is zero. | [Documentation](https://github.com/madgik/exaflow/blob/master/documentation/algorithms/TtestPaired.md) |
| Pearson Correlation | Measures linear association between numerical variables and reports correlations, p-values, and confidence intervals. | [Documentation](https://github.com/madgik/exaflow/blob/master/documentation/algorithms/Pearson.md) |

## Regression and Survival Analysis

| Algorithm | Description | Documentation |
|---|---|---|
| Linear Regression | Fits an ordinary least squares model for a numerical outcome using numerical and/or categorical covariates. | [Documentation](https://github.com/madgik/exaflow/blob/master/documentation/algorithms/LinearRegression.md) |
| Logistic Regression | Models a binary outcome as a function of numerical and/or categorical covariates. | [Documentation](https://github.com/madgik/exaflow/blob/master/documentation/algorithms/LogisticRegression.md) |
| Cox Proportional Hazards Regression | Models time-to-event data with a partial likelihood Cox model and an unspecified baseline hazard. | [Documentation](https://github.com/madgik/exaflow/blob/master/documentation/algorithms/cox_regression_classical.md) |
| Stacked Cox Regression | Approximates time-to-event modeling by expanding survival data into risk-set rows and fitting logistic regression with time-bin indicators. | [Documentation](https://github.com/madgik/exaflow/blob/master/documentation/algorithms/cox_regression_stacked.md) |

## Machine Learning

| Algorithm | Description | Documentation |
|---|---|---|
| K-means | Partitions observations into clusters using numerical variables and squared Euclidean distance. | [Documentation](https://github.com/madgik/exaflow/blob/master/documentation/algorithms/k-means.md) |
| Principal Component Analysis | Computes principal components using global standardization and eigendecomposition of the aggregated covariance matrix. | [Documentation](https://github.com/madgik/exaflow/blob/master/documentation/algorithms/PCA.md) |
| PCA with Transformations | Applies selected per-variable transformations before computing principal components. | [Documentation](https://github.com/madgik/exaflow/blob/master/documentation/algorithms/PCAWithTransformation.md) |
| Gaussian Naive Bayes | Classifies observations with numerical features by modeling each feature as Gaussian within each class. | [Documentation](https://github.com/madgik/exaflow/blob/master/documentation/algorithms/NaiveBayesGaussian.md) |
| Gaussian Naive Bayes K-fold Cross-validation | Evaluates Gaussian Naive Bayes with K-fold cross-validation and returns confusion-matrix and fold-level metrics. | [Documentation](https://github.com/madgik/exaflow/blob/master/documentation/algorithms/NaiveBayesGaussianCV.md) |
| Categorical Naive Bayes | Classifies observations with nominal features by estimating class-conditional category probabilities. | [Documentation](https://github.com/madgik/exaflow/blob/master/documentation/algorithms/NaiveBayesCategorical.md) |
| Categorical Naive Bayes K-fold Cross-validation | Evaluates categorical Naive Bayes with K-fold cross-validation and returns confusion-matrix and fold-level metrics. | [Documentation](https://github.com/madgik/exaflow/blob/master/documentation/algorithms/NaiveBayesCategoricalCV.md) |
| Linear SVM | Fits linear support vector classifiers at each site and averages learned coefficients and intercepts. | [Documentation](https://github.com/madgik/exaflow/blob/master/documentation/algorithms/SVM.md) |
| Federated Averaging | Combines model parameters learned separately at multiple sites by averaging each named parameter array. | [Documentation](https://github.com/madgik/exaflow/blob/master/documentation/algorithms/FederatedAveraging.md) |

## Preprocessing

| Algorithm | Description | Documentation |
|---|---|---|
| Preprocessing Steps | Documents missing-value handling, outlier winsorization, and longitudinal transformation steps used before downstream algorithms. | [Documentation](https://github.com/madgik/exaflow/blob/master/documentation/algorithms/PreprocessingSteps.md) |
| Outlier Winsorizer | Clips selected numerical variables to bounds computed from Gaussian, IQR, MAD, or quantile rules. | [Documentation](https://github.com/madgik/exaflow/blob/master/documentation/algorithms/outlier_winsorizer.md) |
