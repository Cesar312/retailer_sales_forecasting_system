# Project Development Notes

## Docker

### Check PostgreSQL Container Status 

To check the status of all running Docker containers, use the following command in your terminal:
```
docker ps
```

To check the status of the PostgreSQL Container `walmart_postgres`, use the following command in your terminal:
```
docker ps | grep walmart_postgres
```

### Restart PostgreSQL Container
To restart the PostgreSQL container, use the following command in your terminal:
```
docker restart walmart_postgres
```

Docker Compose reads the `docker-compose.yml` file in the current directory to set up and manage multi-container Docker applications. This command ensures your entire service stack matches your docker-compose.yml configuration or after making changes to that file.
```
docker-compose up -d
```

### Stop PostgreSQL Docker Containers
To stop the PostgreSQL container, use the following command in your terminal:
```
docker stop walmart_postgres
```

## Python Virtual Environment

Using Poetry for dependency management and virtual environment. 

### Activate Virtual Environment
To activate the Python virtual environment for this project, use the following command in your terminal:
```
python3.11 -m venv .venv
source .venv/bin/activate
python -m pip install --upgrade pip
```

### Deactivate Virtual Environment
To deactivate the Python virtual environment, simply run:
```
deactivate
```
## Jupyter Notebook

### Start Jupyter Notebook
To start the Jupyter Notebook server, use the following command in your terminal:
```
jupyter notebook
```
This will open the Jupyter Notebook interface in your default web browser.
### Stop Jupyter Notebook
To stop the Jupyter Notebook server, you can interrupt the terminal process by pressing `Ctrl + C`.
## Project Structure
- `notebook/`: Contains Jupyter Notebooks for data analysis and modeling.
- `scripts/`: Python scripts for data processing and model training.
- `venv/`: Python virtual environment directory.
- `docker-compose.yml`: Docker Compose configuration file for setting up services.
- `README.md`: Project documentation and instructions.
- `misc/`: Miscellaneous notes and documentation related to project development.
- `requirements.txt`: List of Python dependencies for the project.
- `.gitignore`: Git ignore file to exclude unnecessary files from version control.
- `config/`: Configuration files for the project.
- `logs/`: Directory for storing log files generated during project execution.
- `tests/`: Unit tests for the project codebase.
- `models/`: Directory for storing trained machine learning models.
- `results/`: Directory for storing output results and visualizations.

## Version Control
This project uses Git for version control. Make sure to commit your changes regularly and push them to the remote repository.
